package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"strand/internal/logger"
	"strand/internal/settings"
	"strand/internal/store"
)

const messagingBridgeName = "messaging"

// ReloadMessaging rebuilds the DB-backed messaging bridge (chat channels).
func (e *Engine) ReloadMessaging() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running || e.store == nil {
		return nil
	}
	return e.reloadMessagingLocked()
}

func (e *Engine) reloadMessagingLocked() error {
	kept := make([]*projectBridge, 0, len(e.bridges))
	for _, b := range e.bridges {
		if b == nil {
			continue
		}
		if b.name == messagingBridgeName {
			_ = b.engine.Stop()
			continue
		}
		kept = append(kept, b)
	}
	e.bridges = kept

	bridge, err := buildMessagingBridge(e.cfg, e.store)
	if err != nil {
		logger.Warnf("messaging bridge: %v", err)
		return err
	}
	if bridge == nil {
		return nil
	}
	if err := bridge.engine.Start(); err != nil {
		_ = bridge.engine.Stop()
		return fmt.Errorf("start messaging bridge: %w", err)
	}
	e.bridges = append(e.bridges, bridge)
	logger.Infof("消息通道已加载: %s platforms=%v", bridge.name, bridge.platforms)
	return nil
}

func buildMessagingBridge(cfg settings.Settings, st *store.Store) (*projectBridge, error) {
	if st == nil {
		return nil, nil
	}
	channels, err := st.Messaging().ListAllEnabledChatChannels()
	if err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return nil, nil
	}

	// Prefer workspace of the first channel's user; fall back to admin.
	userID := channels[0].UserID
	ws, err := st.Messaging().GetWorkspace(userID)
	if err != nil {
		return nil, err
	}
	agentType := strings.TrimSpace(ws.AgentType)
	if agentType == "" {
		return nil, fmt.Errorf("configure messaging workspace agent_type before enabling chat")
	}
	workDir := strings.TrimSpace(ws.WorkDir)
	if workDir == "" {
		workDir = filepath.Join(cfg.Data.Dir, "workspace", "messaging")
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, err
	}

	platforms := make([]map[string]any, 0, len(channels))
	for _, ch := range channels {
		opts, err := ch.PlatformOpts()
		if err != nil {
			_ = st.Messaging().UpdateChannelStatus(ch.ID, "error", err.Error())
			logger.Warnf("messaging channel %s opts: %v", ch.ID, err)
			continue
		}
		opts["type"] = ch.PlatformType
		platforms = append(platforms, opts)
		_ = st.Messaging().UpdateChannelStatus(ch.ID, "starting", "")
	}
	if len(platforms) == 0 {
		return nil, nil
	}

	proj := settings.ProjectConfig{
		Name:     messagingBridgeName,
		WorkDir:  workDir,
		Language: ws.Language,
		Agent: settings.AgentConfig{
			Type:  agentType,
			Model: ws.AgentModel,
		},
		Platforms: platforms,
	}
	bridge, err := buildProject(cfg, proj)
	if err != nil {
		for _, ch := range channels {
			_ = st.Messaging().UpdateChannelStatus(ch.ID, "error", err.Error())
		}
		return nil, err
	}
	for _, ch := range channels {
		_ = st.Messaging().UpdateChannelStatus(ch.ID, "running", "")
	}
	return bridge, nil
}
