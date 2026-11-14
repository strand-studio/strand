package core

import "strings"

// looksLikeAgentAuthRequired detects Claude Code (and similar) login prompts
// that should not be relayed verbatim to chat platforms. Those prompts often
// say "Please run /login", which users mistakenly try as a Telegram command.
func looksLikeAgentAuthRequired(text string) bool {
	s := strings.ToLower(strings.TrimSpace(text))
	if s == "" {
		return false
	}

	notLoggedIn := strings.Contains(s, "not logged in") ||
		strings.Contains(s, "isn't logged in") ||
		strings.Contains(s, "is not logged in") ||
		strings.Contains(s, "haven't logged in") ||
		strings.Contains(s, "have not logged in") ||
		strings.Contains(s, "未登录") ||
		strings.Contains(s, "尚未登录")

	loginHint := strings.Contains(s, "please run /login") ||
		strings.Contains(s, "run /login") ||
		strings.Contains(s, "claude auth login") ||
		strings.Contains(s, "sign in to continue") ||
		strings.Contains(s, "please log in") ||
		strings.Contains(s, "please login")

	if notLoggedIn {
		return true
	}
	// Short agent replies that are essentially a login nudge.
	if loginHint && len(s) < 500 {
		return true
	}
	return false
}

// humanizeAgentAuthMessage replaces agent login prompts with a Strand-facing
// explanation. ok is false when text should be left unchanged.
func (e *Engine) humanizeAgentAuthMessage(text string) (string, bool) {
	if !looksLikeAgentAuthRequired(text) {
		return text, false
	}
	if e == nil || e.i18n == nil {
		return text, false
	}
	return e.i18n.T(MsgAgentNotLoggedIn), true
}
