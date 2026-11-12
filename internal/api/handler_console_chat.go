package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"strand/internal/mcp"
	"strand/internal/store"

	"github.com/gin-gonic/gin"
)

type chatMessageIn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequestBody struct {
	ModelID      string          `json:"model_id"`
	Messages     []chatMessageIn `json:"messages"`
	ToolsEnabled *bool           `json:"tools_enabled"`
	Stream       *bool           `json:"stream"`
	TraderID     string          `json:"trader_id"`
	SessionID    string          `json:"session_id"`
}

func (s *Service) resolveChatModel(userID, modelID string) (*store.AIModel, error) {
	st := s.eng.Store()
	if st == nil {
		return nil, errStoreUnavailable
	}
	want := strings.TrimSpace(modelID)
	if want != "" {
		if m, err := st.AIModel().Get(userID, want); err == nil && m != nil {
			return m, nil
		}
		// FE often passes provider ("claw402") while the row id is "<user>_claw402".
		list, err := st.AIModel().List(userID)
		if err == nil {
			for _, item := range list {
				if item == nil {
					continue
				}
				if item.Provider == want || strings.HasSuffix(item.ID, "_"+want) {
					return item, nil
				}
			}
		}
		return nil, fmt.Errorf("AI model not found — open Config → AI connections to bind one")
	}
	m, err := st.AIModel().GetDefault(userID)
	if err != nil || m == nil {
		return nil, fmt.Errorf("no enabled AI model — open Config → AI connections to bind Claw402 or another provider")
	}
	return m, nil
}

func (s *Service) buildChatClient(model *store.AIModel) (mcp.AIClient, error) {
	provider := strings.TrimSpace(model.Provider)
	if provider == "" {
		provider = "deepseek"
	}
	var client mcp.AIClient
	if provider == "custom" {
		client = mcp.New()
	} else {
		client = mcp.NewAIClientByProvider(provider)
	}
	if client == nil {
		client = mcp.New()
	}
	apiKey := string(model.APIKey)
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("selected model has no API key / wallet key configured")
	}
	switch provider {
	case "claw402":
		client.SetAPIKey(apiKey, "", model.CustomModelName)
	default:
		// Blank custom URL/model must fall back to provider defaults — otherwise
		// SetAPIKey("", "") yields BaseURL="" and requests become "/chat/completions"
		// which SSRF validation rejects (no scheme).
		baseURL, modelName := resolveAIEndpoint(provider, model.CustomAPIURL, model.CustomModelName)
		client.SetAPIKey(apiKey, baseURL, modelName)
	}
	return client, nil
}

func buildChatMessages(in []chatMessageIn, traderID string) []mcp.Message {
	msgs := make([]mcp.Message, 0, len(in)+2)
	sys := chatSystemPrompt
	if traderID != "" {
		sys += "\nPreferred trader_id for account tools: " + traderID
	}
	msgs = append(msgs, mcp.NewSystemMessage(sys))
	for _, m := range in {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		switch role {
		case "assistant":
			msgs = append(msgs, mcp.NewAssistantMessage(content))
		case "system":
			msgs = append(msgs, mcp.NewSystemMessage(content))
		default:
			msgs = append(msgs, mcp.NewUserMessage(content))
		}
	}
	return msgs
}

func writeSSE(c *gin.Context, eventType string, data any) {
	payload, _ := json.Marshal(gin.H{"type": eventType, "data": data})
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(payload)
	_, _ = c.Writer.Write([]byte("\n\n"))
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Service) consoleChatTools(c *gin.Context) {
	defs := chatToolDefs()
	out := make([]gin.H, 0, len(defs))
	for _, t := range defs {
		out = append(out, gin.H{
			"name":        t.Function.Name,
			"description": t.Function.Description,
			"parameters":   t.Function.Parameters,
		})
	}
	writeJSON(c, http.StatusOK, gin.H{"tools": out})
}

func (s *Service) consoleChatToolTest(c *gin.Context) {
	var body struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	raw, _ := json.Marshal(body.Args)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	result := s.executeChatTool(ctx, currentUserID(c), body.Name, string(raw))
	writeJSON(c, http.StatusOK, gin.H{
		"name":   body.Name,
		"result": json.RawMessage(result),
	})
}

func (s *Service) consoleChatCompletions(c *gin.Context) {
	var body chatRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if len(body.Messages) == 0 {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}
	userID := currentUserID(c)
	model, err := s.resolveChatModel(userID, body.ModelID)
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	client, err := s.buildChatClient(model)
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	toolsOn := true
	if body.ToolsEnabled != nil {
		toolsOn = *body.ToolsEnabled
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()
	content, toolTrace, err := s.runChatWithTools(ctx, client, userID, body.TraderID, body.Messages, toolsOn)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{
		"content": content,
		"model": gin.H{
			"id":       model.ID,
			"name":     model.Name,
			"provider": model.Provider,
		},
		"tool_trace": toolTrace,
	})
}

func (s *Service) consoleChatStream(c *gin.Context) {
	var body chatRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if len(body.Messages) == 0 {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}
	userID := currentUserID(c)
	model, err := s.resolveChatModel(userID, body.ModelID)
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	client, err := s.buildChatClient(model)
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	toolsOn := true
	if body.ToolsEnabled != nil {
		toolsOn = *body.ToolsEnabled
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	sessionID := strings.TrimSpace(body.SessionID)
	if sessionID != "" {
		if st := s.eng.Store(); st != nil {
			if _, err := st.ChatSessions().Get(userID, sessionID); err != nil {
				sessionID = ""
			}
		}
	}

	writeSSE(c, "start", gin.H{
		"model":      gin.H{"id": model.ID, "name": model.Name, "provider": model.Provider},
		"session_id": sessionID,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	persistChatTurn := func(assistantText string) {
		if sessionID == "" || s.eng.Store() == nil {
			return
		}
		st := s.eng.Store()
		// Persist only the latest user message in this request.
		for i := len(body.Messages) - 1; i >= 0; i-- {
			if strings.EqualFold(strings.TrimSpace(body.Messages[i].Role), "user") {
				_, _ = st.ChatSessions().AppendMessage(userID, sessionID, "user", body.Messages[i].Content)
				break
			}
		}
		if strings.TrimSpace(assistantText) != "" {
			_, _ = st.ChatSessions().AppendMessage(userID, sessionID, "assistant", assistantText)
		}
	}

	msgs := buildChatMessages(body.Messages, body.TraderID)
	if toolsOn {
		for round := 0; round < 4; round++ {
			req := &mcp.Request{
				Messages:   msgs,
				Tools:      chatToolDefs(),
				ToolChoice: "auto",
				Ctx:        ctx,
			}
			resp, err := client.CallWithRequestFull(req)
			if err != nil {
				writeSSE(c, "error", gin.H{"message": err.Error()})
				writeSSE(c, "done", gin.H{"ok": false})
				return
			}
			if len(resp.ToolCalls) == 0 {
				if resp.Content != "" {
					writeSSE(c, "delta", gin.H{"content": resp.Content})
				}
				persistChatTurn(resp.Content)
				writeSSE(c, "done", gin.H{"ok": true, "session_id": sessionID})
				return
			}
			assistant := mcp.Message{Role: "assistant", ToolCalls: resp.ToolCalls, Content: resp.Content}
			msgs = append(msgs, assistant)
			for _, call := range resp.ToolCalls {
				writeSSE(c, "tool_start", gin.H{"id": call.ID, "name": call.Function.Name, "arguments": call.Function.Arguments})
				result := s.executeChatTool(ctx, userID, call.Function.Name, call.Function.Arguments)
				writeSSE(c, "tool_result", gin.H{"id": call.ID, "name": call.Function.Name, "result": json.RawMessage(result)})
				if sessionID != "" && s.eng.Store() != nil {
					_, _ = s.eng.Store().ChatSessions().AppendToolMessage(
						userID, sessionID, "tool", result, call.Function.Name, call.ID,
					)
				}
				msgs = append(msgs, mcp.Message{
					Role:       "tool",
					ToolCallID: call.ID,
					Content:    result,
				})
			}
		}
	}

	// Final streamed answer (no more tools).
	var builder strings.Builder
	_, err = client.CallWithRequestStream(&mcp.Request{
		Messages: msgs,
		Stream:   true,
		Ctx:      ctx,
	}, func(chunk string) {
		if chunk == "" {
			return
		}
		builder.WriteString(chunk)
		writeSSE(c, "delta", gin.H{"content": chunk})
	})
	if err != nil {
		writeSSE(c, "error", gin.H{"message": err.Error()})
		writeSSE(c, "done", gin.H{"ok": false})
		return
	}
	persistChatTurn(builder.String())
	writeSSE(c, "done", gin.H{"ok": true, "content": builder.String(), "session_id": sessionID})
}

func (s *Service) runChatWithTools(ctx context.Context, client mcp.AIClient, userID, traderID string, in []chatMessageIn, toolsOn bool) (string, []gin.H, error) {
	msgs := buildChatMessages(in, traderID)
	trace := make([]gin.H, 0)
	if toolsOn {
		for round := 0; round < 4; round++ {
			resp, err := client.CallWithRequestFull(&mcp.Request{
				Messages:   msgs,
				Tools:      chatToolDefs(),
				ToolChoice: "auto",
				Ctx:        ctx,
			})
			if err != nil {
				return "", trace, err
			}
			if len(resp.ToolCalls) == 0 {
				return resp.Content, trace, nil
			}
			msgs = append(msgs, mcp.Message{Role: "assistant", ToolCalls: resp.ToolCalls, Content: resp.Content})
			for _, call := range resp.ToolCalls {
				result := s.executeChatTool(ctx, userID, call.Function.Name, call.Function.Arguments)
				trace = append(trace, gin.H{
					"name":      call.Function.Name,
					"arguments": call.Function.Arguments,
					"result":    json.RawMessage(result),
				})
				msgs = append(msgs, mcp.Message{Role: "tool", ToolCallID: call.ID, Content: result})
			}
		}
	}
	text, err := client.CallWithRequest(&mcp.Request{Messages: msgs, Ctx: ctx})
	return text, trace, err
}
