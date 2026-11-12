package mcp

import "strings"

// Provider name constants

const (
	ProviderDeepSeek = "deepseek"
	ProviderOpenAI   = "openai"
	ProviderClaude   = "claude"
	ProviderQwen     = "qwen"
	ProviderGemini   = "gemini"
	ProviderGrok     = "grok"
	ProviderKimi     = "kimi"
	ProviderMiniMax  = "minimax"

	ProviderClaw402 = "claw402"

	DefaultDeepSeekBaseURL = "https://api.deepseek.com"
	DefaultDeepSeekModel   = "deepseek-chat"

	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	DefaultOpenAIModel   = "gpt-4o-mini"

	DefaultQwenBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	DefaultQwenModel   = "qwen3-max"

	DefaultGrokBaseURL = "https://api.x.ai/v1"
	DefaultGrokModel   = "grok-2-latest"

	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultGeminiModel   = "gemini-2.0-flash"

	DefaultKimiBaseURL = "https://api.moonshot.cn/v1"
	DefaultKimiModel   = "moonshot-v1-8k"

	DefaultMiniMaxBaseURL = "https://api.minimax.io/v1"
	DefaultMiniMaxModel   = "MiniMax-M2.7"
)

// ResolveEndpoint fills blank custom URL/model with provider defaults.
func ResolveEndpoint(provider, customURL, customModel string) (baseURL, model string) {
	baseURL = strings.TrimSpace(customURL)
	model = strings.TrimSpace(customModel)
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderDeepSeek:
		if baseURL == "" {
			baseURL = DefaultDeepSeekBaseURL
		}
		if model == "" {
			model = DefaultDeepSeekModel
		}
	case ProviderOpenAI:
		if baseURL == "" {
			baseURL = DefaultOpenAIBaseURL
		}
		if model == "" {
			model = DefaultOpenAIModel
		}
	case ProviderQwen:
		if baseURL == "" {
			baseURL = DefaultQwenBaseURL
		}
		if model == "" {
			model = DefaultQwenModel
		}
	case ProviderGrok:
		if baseURL == "" {
			baseURL = DefaultGrokBaseURL
		}
		if model == "" {
			model = DefaultGrokModel
		}
	case ProviderGemini:
		if baseURL == "" {
			baseURL = DefaultGeminiBaseURL
		}
		if model == "" {
			model = DefaultGeminiModel
		}
	case ProviderKimi:
		if baseURL == "" {
			baseURL = DefaultKimiBaseURL
		}
		if model == "" {
			model = DefaultKimiModel
		}
	case ProviderMiniMax:
		if baseURL == "" {
			baseURL = DefaultMiniMaxBaseURL
		}
		if model == "" {
			model = DefaultMiniMaxModel
		}
	case ProviderClaude:
		if model == "" {
			model = "claude-sonnet-4-5"
		}
		// Claude keeps its own default base URL when custom URL is empty.
	case ProviderClaw402:
		if model == "" {
			model = "gpt-5.6"
		}
	default:
		if baseURL == "" {
			baseURL = DefaultDeepSeekBaseURL
		}
		if model == "" {
			model = DefaultDeepSeekModel
		}
	}
	return baseURL, model
}

func init() {
	registerOpenAICompatible := func(name, baseURL, model string) {
		RegisterProvider(name, func(opts ...ClientOption) AIClient {
			all := append([]ClientOption{
				WithProvider(name),
				WithBaseURL(baseURL),
				WithModel(model),
			}, opts...)
			return NewClient(all...)
		})
	}
	registerOpenAICompatible(ProviderDeepSeek, DefaultDeepSeekBaseURL, DefaultDeepSeekModel)
	registerOpenAICompatible(ProviderOpenAI, DefaultOpenAIBaseURL, DefaultOpenAIModel)
	registerOpenAICompatible(ProviderQwen, DefaultQwenBaseURL, DefaultQwenModel)
	registerOpenAICompatible(ProviderGrok, DefaultGrokBaseURL, DefaultGrokModel)
	registerOpenAICompatible(ProviderGemini, DefaultGeminiBaseURL, DefaultGeminiModel)
	registerOpenAICompatible(ProviderKimi, DefaultKimiBaseURL, DefaultKimiModel)
	registerOpenAICompatible(ProviderMiniMax, DefaultMiniMaxBaseURL, DefaultMiniMaxModel)
}
