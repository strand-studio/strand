package engine

import "strand/internal/core"

type projectBridge struct {
	name      string
	agent     string
	platforms []string
	engine    *core.Engine
}

func projectAgentName(p *projectBridge) string {
	if p == nil {
		return ""
	}
	return p.agent
}

func projectPlatformNames(p *projectBridge) []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.platforms...)
}
