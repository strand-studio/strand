package mcp

import "time"

type ClientEmbedder interface {
	BaseClient() *Client
}

// AIClient public AI client interface (for external use)
type AIClient interface {
	SetAPIKey(apiKey string, customURL string, customModel string)
	SetTimeout(timeout time.Duration)
	CallWithMessages(systemPrompt, userPrompt string) (string, error)
	CallWithRequest(req *Request) (string, error)

	CallWithRequestStream(req *Request, onChunk func(string)) (string, error)

	CallWithRequestFull(req *Request) (*LLMResponse, error)
}
