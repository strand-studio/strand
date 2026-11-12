package engine

// Side-effect imports register platforms and agents with core.
import (
	_ "strand/internal/agent/claudecode"
	_ "strand/internal/agent/codex"
	_ "strand/internal/agent/copilot"
	_ "strand/internal/agent/cursor"
	_ "strand/internal/agent/gemini"
	_ "strand/internal/agent/kimi"
	_ "strand/internal/agent/opencode"

	_ "strand/internal/platform/cloud-web"
	_ "strand/internal/platform/dingtalk"
	_ "strand/internal/platform/discord"
	_ "strand/internal/platform/feishu"
	_ "strand/internal/platform/googlechat"
	_ "strand/internal/platform/line"
	_ "strand/internal/platform/matrix"
	_ "strand/internal/platform/max"
	_ "strand/internal/platform/qq"
	_ "strand/internal/platform/qqbot"
	_ "strand/internal/platform/slack"
	_ "strand/internal/platform/telegram"
	_ "strand/internal/platform/tuitui"
	_ "strand/internal/platform/webex"
	_ "strand/internal/platform/wecom"
	_ "strand/internal/platform/weibo"
	_ "strand/internal/platform/weixin"
	_ "strand/internal/platform/wps-agentspace"
	_ "strand/internal/platform/wps-xiezuo"
	_ "strand/internal/platform/yuanbao"
)
