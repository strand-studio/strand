package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"strand/internal/config"
	"strand/internal/core"
	"strand/internal/logger"
	"strand/internal/manager"
	"strand/internal/notify/fanout"
	telegramnotify "strand/internal/notify/telegram"
	"strand/internal/settings"
	"strand/internal/store"
)

// Engine orchestrates messaging bridges, traders, and shared store.
type Engine struct {
	mu sync.RWMutex

	cfg      settings.Settings
	store    *store.Store
	traders  *manager.TraderManager
	tgBinder *telegramnotify.Binder
	bridges  []*projectBridge
	started  time.Time
	running  bool
	rootCtx  context.Context
	rootCancel context.CancelFunc
}

// NewEngine builds a runtime engine from settings (does not start yet).
func NewEngine(cfg settings.Settings) *Engine {
	return &Engine{cfg: cfg}
}

// UseStoreForTest wires an open store for API tests without full Start().
func (e *Engine) UseStoreForTest(st *store.Store) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store = st
	e.traders = manager.NewTraderManager()
	e.running = true
}

// Settings returns the active settings snapshot.
func (e *Engine) Settings() settings.Settings {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg
}

// Store returns the database handle (nil before Start).
func (e *Engine) Store() *store.Store {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.store
}

// Traders returns the trader manager (nil before Start or when trading disabled).
func (e *Engine) Traders() *manager.TraderManager {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.traders
}

// TelegramBinder returns the console Telegram binder (may be nil).
func (e *Engine) TelegramBinder() *telegramnotify.Binder {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tgBinder
}

// NotifyTelegram pushes a plain-text alert to the bound console chat.
func (e *Engine) NotifyTelegram(text string) {
	b := e.TelegramBinder()
	if b == nil {
		return
	}
	if err := b.Notify(text); err != nil {
		logger.Debugf("telegram notify skipped: %v", err)
	}
}

// RestartTelegramBinder reloads the console bot poller with a new token.
func (e *Engine) RestartTelegramBinder(token string) {
	e.mu.RLock()
	binder := e.tgBinder
	ctx := e.rootCtx
	e.mu.RUnlock()
	if binder == nil || ctx == nil {
		return
	}
	binder.Restart(ctx, token)
}

// Start opens the database, messaging projects, and optional auto-traders.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("engine already running")
	}

	if err := os.MkdirAll(e.cfg.Data.Dir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	_ = logger.Init(&logger.Config{Level: e.cfg.Runtime.LogLevel})

	if err := config.Apply(e.cfg); err != nil {
		return err
	}

	st, err := openStore(e.cfg)
	if err != nil {
		return err
	}
	e.store = st

	if e.cfg.Trading.Enabled {
		tm := manager.NewTraderManager()
		if err := tm.LoadTradersFromStore(st); err != nil {
			logger.Warnf("加载交易员失败: %v", err)
		} else if e.cfg.Runtime.AutoStartTraders {
			tm.AutoStartRunningTraders(st)
		}
		e.traders = tm
	}

	bridges, err := startProjects(e.cfg)
	if err != nil {
		_ = st.Close()
		e.store = nil
		e.traders = nil
		return err
	}
	e.bridges = bridges
	e.rootCtx, e.rootCancel = context.WithCancel(context.Background())
	e.tgBinder = telegramnotify.NewBinder(st)
	e.tgBinder.Start(e.rootCtx)
	if e.traders != nil {
		alerter := fanout.New(st, e.tgBinder)
		e.traders.SetNotifyAlert(func(text string) {
			alerter.Notify(text)
		})
	}
	e.started = time.Now()
	e.running = true

	logger.Infof("strand engine ready (projects=%d trading=%v api=%v)",
		len(bridges), e.cfg.Trading.Enabled, e.cfg.API.Enabled)
	return nil
}

// Stop shuts down bridges, traders, and the database.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	var first error
	capture := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}

	if e.tgBinder != nil {
		e.tgBinder.Stop()
		e.tgBinder = nil
	}
	if e.rootCancel != nil {
		e.rootCancel()
		e.rootCancel = nil
	}
	for _, b := range e.bridges {
		if b != nil && b.engine != nil {
			capture(b.engine.Stop())
		}
	}
	e.bridges = nil

	if e.traders != nil {
		e.traders.StopAll()
		e.traders = nil
	}

	if e.store != nil {
		capture(e.store.Close())
		e.store = nil
	}

	e.running = false
	logger.Info("strand engine stopped")
	return first
}

// Status is a JSON-friendly runtime snapshot for CLI / API.
type Status struct {
	Running   bool              `json:"running"`
	StartedAt string            `json:"started_at,omitempty"`
	UptimeSec int64             `json:"uptime_sec,omitempty"`
	DataDir   string            `json:"data_dir"`
	Config    string            `json:"config,omitempty"`
	API       StatusAPI         `json:"api"`
	Database  StatusDatabase    `json:"database"`
	Trading   StatusTrading     `json:"trading"`
	Projects  []StatusProject   `json:"projects"`
	Agents    []string          `json:"registered_agents"`
	Platforms []string          `json:"registered_platforms"`
}

type StatusAPI struct {
	Enabled bool   `json:"enabled"`
	Addr    string `json:"addr"`
}

type StatusDatabase struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

type StatusTrading struct {
	Enabled bool `json:"enabled"`
	Loaded  int  `json:"loaded"`
	Running int  `json:"running"`
}

type StatusProject struct {
	Name      string   `json:"name"`
	Agent     string   `json:"agent"`
	Platforms []string `json:"platforms"`
}

// Snapshot returns the current runtime status.
func (e *Engine) Snapshot() Status {
	e.mu.RLock()
	defer e.mu.RUnlock()

	st := Status{
		Running:   e.running,
		DataDir:   e.cfg.Data.Dir,
		Config:    e.cfg.Path,
		API:       StatusAPI{Enabled: e.cfg.API.Enabled, Addr: e.cfg.Addr()},
		Database:  StatusDatabase{Type: e.cfg.Database.Type, Path: e.cfg.Database.Path},
		Trading:   StatusTrading{Enabled: e.cfg.Trading.Enabled},
		Agents:    core.ListRegisteredAgents(),
		Platforms: core.ListRegisteredPlatforms(),
		Projects:  make([]StatusProject, 0, len(e.bridges)),
	}
	if e.running && !e.started.IsZero() {
		st.StartedAt = e.started.UTC().Format(time.RFC3339)
		st.UptimeSec = int64(time.Since(e.started).Seconds())
	}
	if e.traders != nil {
		all := e.traders.GetAllTraders()
		st.Trading.Loaded = len(all)
		for _, t := range all {
			status := t.GetStatus()
			if running, _ := status["is_running"].(bool); running {
				st.Trading.Running++
			}
		}
	}
	for _, b := range e.bridges {
		st.Projects = append(st.Projects, StatusProject{
			Name:      b.name,
			Agent:     projectAgentName(b),
			Platforms: projectPlatformNames(b),
		})
	}
	// Also surface configured-but-not-started projects from settings when not running.
	if !e.running {
		for _, p := range e.cfg.Projects {
			plats := make([]string, 0, len(p.Platforms))
			for _, raw := range p.Platforms {
				if t, _ := raw["type"].(string); t != "" {
					plats = append(plats, t)
				}
			}
			st.Projects = append(st.Projects, StatusProject{
				Name:      p.Name,
				Agent:     p.Agent.Type,
				Platforms: plats,
			})
		}
	}
	return st
}

func openStore(cfg settings.Settings) (*store.Store, error) {
	dbCfg := store.DBConfig{
		Type:     store.DBType(cfg.Database.Type),
		Path:     cfg.Database.Path,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}
	if dbCfg.Type == store.DBTypeSQLite {
		if err := os.MkdirAll(filepath.Dir(dbCfg.Path), 0o755); err != nil {
			return nil, err
		}
	}
	return store.NewWithConfig(dbCfg)
}

func startProjects(cfg settings.Settings) ([]*projectBridge, error) {
	if len(cfg.Projects) == 0 {
		logger.Info("未配置 [[projects]]，仅启动 API / 交易（如已启用）")
		return nil, nil
	}

	out := make([]*projectBridge, 0, len(cfg.Projects))
	for _, p := range cfg.Projects {
		bridge, err := buildProject(cfg, p)
		if err != nil {
			for _, started := range out {
				_ = started.engine.Stop()
			}
			return nil, fmt.Errorf("项目 %q: %w", p.Name, err)
		}
		if err := bridge.engine.Start(); err != nil {
			_ = bridge.engine.Stop()
			for _, started := range out {
				_ = started.engine.Stop()
			}
			return nil, fmt.Errorf("项目 %q 启动失败: %w", p.Name, err)
		}
		out = append(out, bridge)
		logger.Infof("消息项目已启动: %s", p.Name)
	}
	return out, nil
}

func buildProject(cfg settings.Settings, p settings.ProjectConfig) (*projectBridge, error) {
	if p.Agent.Type == "" {
		return nil, fmt.Errorf("缺少 agent.type")
	}
	if len(p.Platforms) == 0 {
		return nil, fmt.Errorf("至少配置一个 platform")
	}

	if err := os.MkdirAll(p.WorkDir, 0o755); err != nil {
		return nil, err
	}
	sessionDir := cfg.SessionDir(p.Name)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}

	agentOpts := map[string]any{}
	for k, v := range p.Agent.Options {
		agentOpts[k] = v
	}
	if _, ok := agentOpts["work_dir"]; !ok {
		agentOpts["work_dir"] = p.WorkDir
	}
	if p.Agent.Model != "" {
		agentOpts["model"] = p.Agent.Model
	}
	agentOpts["cc_data_dir"] = cfg.Data.Dir
	agentOpts["cc_project"] = p.Name

	ag, err := core.CreateAgent(p.Agent.Type, agentOpts)
	if err != nil {
		return nil, err
	}

	platformNames := make([]string, 0, len(p.Platforms))
	platforms := make([]core.Platform, 0, len(p.Platforms))
	for _, raw := range p.Platforms {
		typ, _ := raw["type"].(string)
		if typ == "" {
			return nil, fmt.Errorf("platform 缺少 type")
		}
		opts := map[string]any{}
		for k, v := range raw {
			if k == "type" {
				continue
			}
			opts[k] = v
		}
		opts["cc_data_dir"] = cfg.Data.Dir
		opts["cc_project"] = p.Name
		if _, ok := opts["work_dir"]; !ok {
			opts["work_dir"] = p.WorkDir
		}
		plat, err := core.CreatePlatform(typ, opts)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, plat)
		platformNames = append(platformNames, typ)
	}

	lang := core.NormalizeLanguageString(p.Language)
	sessionPath := filepath.Join(sessionDir, "sessions.json")
	eng := core.NewEngine(p.Name, ag, platforms, sessionPath, lang)
	return &projectBridge{
		name:      p.Name,
		agent:     p.Agent.Type,
		platforms: platformNames,
		engine:    eng,
	}, nil
}
