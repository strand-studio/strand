package mcp

import "testing"

func TestUsesMaxCompletionTokens(t *testing.T) {
	cases := []struct {
		provider, model, base string
		want                  bool
	}{
		{ProviderOpenAI, "gpt-4o-mini", DefaultOpenAIBaseURL, true},
		{ProviderCustom, "gpt-4o-mini", DefaultOpenAIBaseURL, true},
		{ProviderCustom, "gpt-5.6", "https://relay.example/v1", true},
		{ProviderCustom, "o3-mini", "https://relay.example/v1", true},
		{ProviderDeepSeek, "deepseek-chat", DefaultDeepSeekBaseURL, false},
		{ProviderCustom, "deepseek-chat", DefaultDeepSeekBaseURL, false},
	}
	for _, tc := range cases {
		got := usesMaxCompletionTokens(tc.provider, tc.model, tc.base)
		if got != tc.want {
			t.Fatalf("%s/%s/%s: got %v want %v", tc.provider, tc.model, tc.base, got, tc.want)
		}
	}
}

func TestReasoningEffortNoneWhenToolsOnGPT6(t *testing.T) {
	c := NewAIClientByProvider(ProviderOpenAI).(*Client)
	c.SetAPIKey("sk-test", DefaultOpenAIBaseURL, "gpt-6-luna")
	body := c.BuildRequestBodyFromRequest(&Request{
		Model: "gpt-6-luna",
		Messages: []Message{
			NewUserMessage("hi"),
		},
		Tools: []Tool{{
			Type: "function",
			Function: FunctionDef{
				Name:       "list_markets",
				Parameters: map[string]any{"type": "object"},
			},
		}},
		ToolChoice: "auto",
	})
	if body["reasoning_effort"] != "none" {
		t.Fatalf("reasoning_effort=%v want none; body=%#v", body["reasoning_effort"], body)
	}
}

func TestSetAPIKeyKeepsOpenAIProviderForDefaultURL(t *testing.T) {
	c := NewAIClientByProvider(ProviderOpenAI).(*Client)
	c.SetAPIKey("sk-test", DefaultOpenAIBaseURL, DefaultOpenAIModel)
	if c.Provider != ProviderOpenAI {
		t.Fatalf("provider=%q want %q", c.Provider, ProviderOpenAI)
	}
	body := c.BuildMCPRequestBody("sys", "hi")
	if _, ok := body["max_completion_tokens"]; !ok {
		t.Fatalf("expected max_completion_tokens in body: %#v", body)
	}
	if _, ok := body["max_tokens"]; ok {
		t.Fatalf("did not expect max_tokens in body: %#v", body)
	}
}
