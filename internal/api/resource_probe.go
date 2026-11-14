package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"strand/internal/store"
)

func probeTelegramToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	trimmed := strings.TrimSpace(string(body))
	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 &&
			(strings.Contains(trimmed, `"ok":true`) || strings.Contains(trimmed, `"ok": true`)) {
			// Extremely unlikely truncation on getMe; accept clear success prefix.
			return nil
		}
		return fmt.Errorf("telegram getMe returned non-JSON (%d): %s", resp.StatusCode, truncateRunes(trimmed, 160))
	}
	if resp.StatusCode >= 300 || !parsed.OK {
		msg := strings.TrimSpace(parsed.Description)
		if msg == "" {
			msg = truncateRunes(trimmed, 160)
		}
		return fmt.Errorf("telegram getMe failed: %s", msg)
	}
	return nil
}

func probeDiscordToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	trimmed := strings.TrimSpace(string(body))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord probe http %d: %s", resp.StatusCode, truncateRunes(trimmed, 160))
	}
	// Require a real Discord user payload — bare HTTP 200 HTML must not pass.
	var parsed struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil ||
		(strings.TrimSpace(parsed.ID) == "" && strings.TrimSpace(parsed.Username) == "") {
		if strings.Contains(trimmed, `"id"`) && strings.Contains(trimmed, `"username"`) {
			return nil
		}
		return fmt.Errorf("discord probe returned HTTP %d but not a bot user payload: %s",
			resp.StatusCode, truncateRunes(trimmed, 160))
	}
	return nil
}

func probeSlackToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/auth.test", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	trimmed := strings.TrimSpace(string(body))
	var parsed struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 &&
			(strings.Contains(trimmed, `"ok":true`) || strings.Contains(trimmed, `"ok": true`)) {
			return nil
		}
		return fmt.Errorf("slack auth.test returned non-JSON (%d): %s", resp.StatusCode, truncateRunes(trimmed, 120))
	}
	if resp.StatusCode >= 300 || !parsed.OK {
		return fmt.Errorf("slack auth.test failed: %s", firstNonEmpty(parsed.Error, truncateRunes(trimmed, 120)))
	}
	return nil
}

func probePlatformChannel(ch *store.MessagingChannel) error {
	if ch == nil {
		return fmt.Errorf("channel required")
	}
	creds, err := ch.CredentialMap()
	if err != nil {
		return err
	}
	switch strings.ToLower(ch.PlatformType) {
	case "telegram":
		return probeTelegramToken(creds["token"])
	case "discord":
		if wh := strings.TrimSpace(creds["webhook_url"]); wh != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, wh, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("discord webhook http %d", resp.StatusCode)
			}
			return nil
		}
		return probeDiscordToken(creds["token"])
	case "slack":
		if wh := strings.TrimSpace(creds["webhook_url"]); wh != "" {
			return nil // webhook presence is enough for soft probe
		}
		return probeSlackToken(creds["token"])
	case "weixin":
		if strings.TrimSpace(creds["token"]) == "" {
			return fmt.Errorf("weixin token required")
		}
		// Token format check only — full ilink poll needs long-lived session.
		if len(strings.TrimSpace(creds["token"])) < 8 {
			return fmt.Errorf("weixin token looks too short")
		}
		return nil
	default:
		if len(creds) == 0 {
			return fmt.Errorf("no credentials configured")
		}
		return nil
	}
}

var agentBinaries = map[string][]string{
	"codex":      {"codex"},
	"claudecode": {"claude", "claude-code"},
	"cursor":     {"cursor-agent", "agent"},
	"gemini":     {"gemini"},
	"kimi":       {"kimi"},
	"opencode":   {"opencode"},
	"copilot":    {"copilot"},
}

func probeAgent(agentType string) error {
	agentType = strings.TrimSpace(strings.ToLower(agentType))
	if agentType == "" {
		return fmt.Errorf("agent_type required")
	}
	bins := agentBinaries[agentType]
	if len(bins) == 0 {
		bins = []string{agentType}
	}
	var tried []string
	for _, bin := range bins {
		path, err := exec.LookPath(bin)
		if err != nil {
			tried = append(tried, bin)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		cmd := exec.CommandContext(ctx, path, "--version")
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("%s timed out on --version", bin)
			}
			_ = out
			// Binary exists but --version failed: still continue to auth checks
			// when we know how; otherwise treat as soft-found.
		}
		if err := probeAgentReady(agentType, path); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("agent binary not found on PATH (tried: %s)", strings.Join(tried, ", "))
}

// probeAgentReady checks that a found agent binary is actually usable
// (auth / session), not merely present on PATH.
func probeAgentReady(agentType, binPath string) error {
	switch agentType {
	case "claudecode":
		return probeClaudeCodeAuth(binPath)
	default:
		return nil
	}
}

func probeClaudeCodeAuth(binPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, "auth", "status")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return fmt.Errorf("claude auth status timed out")
	}
	trimmed := strings.TrimSpace(string(out))
	if err != nil && trimmed == "" {
		return fmt.Errorf("claude auth status failed: %v (run `claude auth login` on this machine)", err)
	}
	loggedIn, parseErr := parseClaudeAuthLoggedIn(trimmed)
	if parseErr != nil {
		// Older CLIs may print prose instead of JSON.
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		lower := strings.ToLower(trimmed + " " + errText)
		if strings.Contains(lower, "not logged") ||
			strings.Contains(lower, `loggedin":false`) ||
			strings.Contains(lower, `"loggedin": false`) ||
			(strings.Contains(lower, "please run") && strings.Contains(lower, "login")) {
			return fmt.Errorf("Claude Code is not logged in; run `claude auth login` on this machine")
		}
		if err != nil {
			return fmt.Errorf("claude auth status failed: %v", err)
		}
		// Ambiguous output but command succeeded — do not fake OK.
		return fmt.Errorf("could not verify Claude Code login (%v); run `claude auth status`", parseErr)
	}
	if !loggedIn {
		return fmt.Errorf("Claude Code is not logged in; run `claude auth login` on this machine")
	}
	return nil
}

func parseClaudeAuthLoggedIn(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, fmt.Errorf("empty auth status")
	}
	// Prefer a JSON object in the output (may be surrounded by log lines).
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		var parsed struct {
			LoggedIn bool `json:"loggedIn"`
		}
		if err := json.Unmarshal([]byte(raw[start:end+1]), &parsed); err == nil {
			return parsed.LoggedIn, nil
		}
	}
	return false, fmt.Errorf("unrecognized auth status format")
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
