// Package settings is the shared process configuration loaded from TOML.
// Secrets and deployment overrides come from the environment / .env.
package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Settings is what the CLI and the running service share.
type Settings struct {
	Data     DataConfig     `toml:"data"`
	API      APIConfig      `toml:"api"`
	Database DatabaseConfig `toml:"database"`
	Runtime  RuntimeConfig  `toml:"runtime"`
	Trading  TradingConfig  `toml:"trading"`
	Market   MarketConfig   `toml:"market"`
	Projects []ProjectConfig `toml:"projects"`

	// Path is the absolute path of the loaded TOML file (not from file).
	Path string `toml:"-"`
}

type DataConfig struct {
	Dir string `toml:"dir"`
}

type APIConfig struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	Token   string `toml:"token"` // prefer STRAND_API_TOKEN / env expansion
	Mode    string `toml:"mode"`  // gin mode: debug | release | test
}

type DatabaseConfig struct {
	Type     string `toml:"type"` // sqlite | postgres
	Path     string `toml:"path"` // sqlite relative to data.dir unless absolute
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Name     string `toml:"name"`
	SSLMode  string `toml:"sslmode"`
}

type RuntimeConfig struct {
	Language               string `toml:"language"` // zh | en | ...
	LogLevel               string `toml:"log_level"`
	ExperienceImprovement  bool   `toml:"experience_improvement"`
	AutoStartTraders       bool   `toml:"auto_start_traders"`
}

type TradingConfig struct {
	Enabled bool `toml:"enabled"`
}

type MarketConfig struct {
	AlpacaAPIKey    string `toml:"alpaca_api_key"`
	AlpacaSecretKey string `toml:"alpaca_secret_key"`
	TwelveDataKey   string `toml:"twelve_data_key"`
}

// ProjectConfig is one messaging workspace (agent + platforms).
type ProjectConfig struct {
	Name      string              `toml:"name"`
	WorkDir   string              `toml:"work_dir"`
	Language  string              `toml:"language"`
	Agent     AgentConfig         `toml:"agent"`
	Platforms []map[string]any    `toml:"platforms"`
}

type AgentConfig struct {
	Type    string         `toml:"type"`
	Model   string         `toml:"model"`
	Options map[string]any `toml:"options"`
}

// Default returns a usable empty settings object (before Load).
func Default() Settings {
	dir := DefaultDataDir()
	return Settings{
		Data: DataConfig{Dir: dir},
		API: APIConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    8080,
			Mode:    "release",
		},
		Database: DatabaseConfig{
			Type:    "sqlite",
			Path:    "data.db",
			Host:    "localhost",
			Port:    5432,
			User:    "postgres",
			Name:    "strand",
			SSLMode: "disable",
		},
		Runtime: RuntimeConfig{
			Language:              "zh",
			LogLevel:              "info",
			ExperienceImprovement: true,
			AutoStartTraders:      true,
		},
		Trading: TradingConfig{Enabled: true},
		Market:  MarketConfig{},
	}
}

// Load reads TOML from path (or the default search path), expands ${ENV},
// then applies environment overrides.
func Load(path string) (Settings, error) {
	cfg := Default()

	resolved, err := ResolveConfigPath(path)
	if err != nil {
		return cfg, err
	}

	// Soft-load .env next to config / cwd / data dir before expanding.
	loadDotEnvCandidates(resolved)

	if resolved != "" {
		if err := decodeFile(resolved, &cfg); err != nil {
			return cfg, err
		}
		cfg.Path = resolved
	}

	expandSettings(&cfg)
	applyEnvOverrides(&cfg)
	normalize(&cfg)
	return cfg, nil
}

// MustExistLoad fails if no config file is found.
func MustExistLoad(path string) (Settings, error) {
	resolved, err := ResolveConfigPath(path)
	if err != nil {
		return Settings{}, err
	}
	if resolved == "" {
		return Settings{}, fmt.Errorf("找不到配置文件，请先运行: strand config init")
	}
	if _, err := os.Stat(resolved); err != nil {
		return Settings{}, fmt.Errorf("配置文件不存在: %s（请先运行 strand config init）", resolved)
	}
	return Load(resolved)
}

func normalize(cfg *Settings) {
	cfg.Data.Dir = ExpandHome(strings.TrimSpace(cfg.Data.Dir))
	if cfg.Data.Dir == "" {
		cfg.Data.Dir = DefaultDataDir()
	}
	if cfg.API.Port <= 0 {
		cfg.API.Port = 8080
	}
	if cfg.API.Host == "" {
		cfg.API.Host = "127.0.0.1"
	}
	cfg.API.Mode = NormalizeGinMode(cfg.API.Mode)
	if cfg.Database.Type == "" {
		cfg.Database.Type = "sqlite"
	}
	cfg.Database.Type = strings.ToLower(cfg.Database.Type)
	if cfg.Database.Path == "" {
		cfg.Database.Path = "data.db"
	}
	if !filepath.IsAbs(cfg.Database.Path) && cfg.Database.Type == "sqlite" {
		cfg.Database.Path = filepath.Join(cfg.Data.Dir, cfg.Database.Path)
	}
	if cfg.Runtime.Language == "" {
		cfg.Runtime.Language = "zh"
	}
	if cfg.Runtime.LogLevel == "" {
		cfg.Runtime.LogLevel = "info"
	}
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		if p.Name == "" {
			p.Name = fmt.Sprintf("project-%d", i+1)
		}
		p.WorkDir = ExpandHome(p.WorkDir)
		if p.WorkDir == "" {
			p.WorkDir = filepath.Join(cfg.Data.Dir, "workspace", p.Name)
		}
		if p.Language == "" {
			p.Language = cfg.Runtime.Language
		}
	}
}

// Addr returns host:port for the HTTP API.
func (s Settings) Addr() string {
	return fmt.Sprintf("%s:%d", s.API.Host, s.API.Port)
}

// NormalizeGinMode maps a config string to gin's mode name.
// Accepted: debug, release, test (case-insensitive). Empty / unknown → release.
func NormalizeGinMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "debug", "dev", "development":
		return "debug"
	case "test":
		return "test"
	case "release", "prod", "production", "":
		return "release"
	default:
		return "release"
	}
}

// LogPath is where a detached server writes stdout.
func (s Settings) LogPath() string {
	return filepath.Join(s.Data.Dir, "strand.log")
}

// ProcPath is the pid file of the running server.
func (s Settings) ProcPath() string {
	return filepath.Join(s.Data.Dir, "strand.pid")
}

// SessionDir is where messaging session state is stored.
func (s Settings) SessionDir(project string) string {
	return filepath.Join(s.Data.Dir, "sessions", project)
}

// Mask keeps a secret recognizable without printing it.
func Mask(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
