package messaging

import "strand/internal/core"

// Field describes one credential / option shown in Web or CLI.
type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Secret      bool   `json:"secret,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

// PlatformSchema is the UI contract for adding a channel.
type PlatformSchema struct {
	Type                 string  `json:"type"`
	Label                string  `json:"label"`
	Description          string  `json:"description"`
	Fields               []Field `json:"fields"`
	Chat                 bool    `json:"chat"`
	Alerts               bool    `json:"alerts"`
	BoundPeerPlaceholder string  `json:"bound_peer_placeholder,omitempty"`
	BoundPeerHelp        string  `json:"bound_peer_help,omitempty"`
}

var catalog = []PlatformSchema{
	{
		Type: "telegram", Label: "Telegram", Description: "Bot conversation and trade alerts",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "Telegram chat id (e.g. 123456789)",
		BoundPeerHelp: "Numeric chat id that should receive trade alerts",
		Fields: []Field{
			{Key: "token", Label: "Bot token", Secret: true, Required: true, Placeholder: "123456:ABC…", Help: "From @BotFather"},
		},
	},
	{
		Type: "weixin", Label: "微信", Description: "个人微信 ilink 对话与通知",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "微信用户 wxid（可选）",
		BoundPeerHelp: "告警推送到该微信用户；留空则按对话上下文",
		Fields: []Field{
			{Key: "token", Label: "ilink Token", Secret: true, Required: true, Help: "微信 ilink bot Bearer token"},
			{Key: "allow_from", Label: "Allow from", Required: false, Placeholder: "wxid_…", Help: "可选白名单，逗号分隔"},
		},
	},
	{
		Type: "wecom", Label: "企业微信", Description: "企业微信应用对话",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "企业微信 UserID / 部门",
		BoundPeerHelp: "告警接收人的企微 UserID",
		Fields: []Field{
			{Key: "corp_id", Label: "Corp ID", Required: true},
			{Key: "corp_secret", Label: "Corp Secret", Secret: true, Required: true},
			{Key: "agent_id", Label: "Agent ID", Required: true},
			{Key: "token", Label: "Callback Token", Secret: true, Required: false},
			{Key: "encoding_aes_key", Label: "EncodingAESKey", Secret: true, Required: false},
		},
	},
	{
		Type: "discord", Label: "Discord", Description: "Bot conversation and alerts",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "Discord channel id",
		BoundPeerHelp: "Numeric channel id for alert messages",
		Fields: []Field{
			{Key: "token", Label: "Bot token", Secret: true, Required: true},
			{Key: "webhook_url", Label: "Webhook URL (alerts)", Secret: true, Required: false},
		},
	},
	{
		Type: "slack", Label: "Slack", Description: "Bot conversation and alerts",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "Slack channel id (e.g. C0123ABC)",
		BoundPeerHelp: "Channel that should receive trade alerts",
		Fields: []Field{
			{Key: "token", Label: "Bot token", Secret: true, Required: true},
			{Key: "webhook_url", Label: "Webhook URL (alerts)", Secret: true, Required: false},
		},
	},
	{
		Type: "feishu", Label: "飞书", Description: "飞书应用机器人",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "飞书 chat_id / open_id",
		BoundPeerHelp: "告警推送到的飞书会话或用户",
		Fields: []Field{
			{Key: "app_id", Label: "App ID", Required: true},
			{Key: "app_secret", Label: "App Secret", Secret: true, Required: true},
		},
	},
	{
		Type: "lark", Label: "Lark", Description: "Lark (international Feishu)",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "Lark chat_id / open_id",
		BoundPeerHelp: "Chat or user that should receive alerts",
		Fields: []Field{
			{Key: "app_id", Label: "App ID", Required: true},
			{Key: "app_secret", Label: "App Secret", Secret: true, Required: true},
		},
	},
	{
		Type: "dingtalk", Label: "钉钉", Description: "钉钉机器人",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "钉钉 conversation / user id",
		BoundPeerHelp: "告警目标会话或用户",
		Fields: []Field{
			{Key: "client_id", Label: "Client ID", Required: true},
			{Key: "client_secret", Label: "Client Secret", Secret: true, Required: true},
		},
	},
	{
		Type: "qq", Label: "QQ", Description: "QQ 通道",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "QQ 号 / 群号",
		BoundPeerHelp: "告警推送到的 QQ 用户或群",
		Fields: []Field{
			{Key: "app_id", Label: "App ID", Required: true},
			{Key: "token", Label: "Token", Secret: true, Required: true},
			{Key: "app_secret", Label: "App Secret", Secret: true, Required: false},
		},
	},
	{
		Type: "qqbot", Label: "QQ Bot", Description: "QQ 官方机器人",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "QQ 号 / 群号",
		BoundPeerHelp: "告警推送到的 QQ 用户或群",
		Fields: []Field{
			{Key: "app_id", Label: "App ID", Required: true},
			{Key: "secret", Label: "Secret", Secret: true, Required: true},
		},
	},
	{
		Type: "line", Label: "LINE", Description: "LINE Messaging API",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "LINE user / group id",
		BoundPeerHelp: "User or group that should receive alerts",
		Fields: []Field{
			{Key: "channel_secret", Label: "Channel Secret", Secret: true, Required: true},
			{Key: "channel_token", Label: "Channel Access Token", Secret: true, Required: true},
		},
	},
	{
		Type: "matrix", Label: "Matrix", Description: "Matrix homeserver bot",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "!room:server or @user:server",
		BoundPeerHelp: "Matrix room or user id for alerts",
		Fields: []Field{
			{Key: "homeserver", Label: "Homeserver", Required: true, Placeholder: "https://matrix.org"},
			{Key: "user_id", Label: "User ID", Required: true},
			{Key: "access_token", Label: "Access Token", Secret: true, Required: true},
		},
	},
	{
		Type: "googlechat", Label: "Google Chat", Description: "Google Chat app",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "Google Chat space name",
		BoundPeerHelp: "Space that should receive alerts",
		Fields: []Field{
			{Key: "credentials_json", Label: "Service account JSON", Secret: true, Required: true},
		},
	},
	{
		Type: "webex", Label: "Webex", Description: "Cisco Webex bot",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "Webex room / person id",
		BoundPeerHelp: "Room or person that should receive alerts",
		Fields: []Field{
			{Key: "token", Label: "Bot token", Secret: true, Required: true},
		},
	},
	{
		Type: "weibo", Label: "微博", Description: "微博开放平台",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "微博用户 UID",
		BoundPeerHelp: "告警目标用户",
		Fields: []Field{
			{Key: "access_token", Label: "Access Token", Secret: true, Required: true},
		},
	},
	{
		Type: "tuitui", Label: "推推", Description: "推推消息通道",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "推推用户 / 会话 id",
		BoundPeerHelp: "告警目标",
		Fields: []Field{
			{Key: "token", Label: "Token", Secret: true, Required: true},
		},
	},
	{
		Type: "max", Label: "MAX", Description: "MAX messenger",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "MAX chat / user id",
		BoundPeerHelp: "Alert target chat or user",
		Fields: []Field{
			{Key: "token", Label: "Token", Secret: true, Required: true},
		},
	},
	{
		Type: "yuanbao", Label: "元宝", Description: "腾讯元宝",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "元宝用户 / 会话 id",
		BoundPeerHelp: "告警目标",
		Fields: []Field{
			{Key: "token", Label: "Token", Secret: true, Required: true},
		},
	},
	{
		Type: "wps-xiezuo", Label: "WPS 协作", Description: "WPS 协作机器人",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "WPS 协作会话 id",
		BoundPeerHelp: "告警目标会话",
		Fields: []Field{
			{Key: "token", Label: "Token", Secret: true, Required: true},
		},
	},
	{
		Type: "wps-agentspace", Label: "WPS AgentSpace", Description: "WPS AgentSpace",
		Chat: true, Alerts: true,
		BoundPeerPlaceholder: "AgentSpace 会话 id",
		BoundPeerHelp: "告警目标会话",
		Fields: []Field{
			{Key: "token", Label: "Token", Secret: true, Required: true},
		},
	},
	{
		Type: "cloud_web", Label: "Cloud Web", Description: "Strand cloud web bridge",
		Chat: true, Alerts: false,
		BoundPeerPlaceholder: "Optional peer id",
		Fields: []Field{
			{Key: "token", Label: "Token", Secret: true, Required: true},
		},
	},
}

// SupportedPlatforms returns schemas for platforms that are registered in-process.
func SupportedPlatforms() []PlatformSchema {
	registered := map[string]bool{}
	for _, name := range core.ListRegisteredPlatforms() {
		registered[name] = true
	}
	out := make([]PlatformSchema, 0, len(catalog))
	for _, p := range catalog {
		if registered[p.Type] {
			out = append(out, p)
		}
	}
	// Also surface any registered platform not in the curated catalog.
	known := map[string]bool{}
	for _, p := range out {
		known[p.Type] = true
	}
	for name := range registered {
		if known[name] {
			continue
		}
		out = append(out, PlatformSchema{
			Type: name, Label: name, Description: name,
			Chat: true, Alerts: true,
			Fields: []Field{{Key: "token", Label: "Token", Secret: true, Required: true}},
		})
	}
	return out
}

// Agents returns registered agent type names for the workspace picker.
func Agents() []string {
	return core.ListRegisteredAgents()
}
