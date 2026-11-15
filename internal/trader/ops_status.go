package trader

import (
	"fmt"
	"strings"
)

// FormatOpsStatusLine builds one Telegram /status line from GetStatus (+ optional equity).
// Hard-gate pause (pause_reason / stop_until) is shown separately from AI Safe Mode.
func FormatOpsStatusLine(status map[string]interface{}, equity float64, equityOK bool) string {
	if status == nil {
		return "• (unknown)"
	}
	name := strings.TrimSpace(asString(status["trader_name"]))
	if name == "" {
		name = strings.TrimSpace(asString(status["trader_id"]))
	}
	if name == "" {
		name = "Autopilot"
	}

	parts := []string{"• " + name}
	if equityOK && equity > 0 {
		parts = append(parts, fmt.Sprintf("equity $%.2f", equity))
	}

	dayStart := asFloat(status["day_start_equity"])
	peak := asFloat(status["peak_equity"])
	dayLimit := asFloat(status["daily_loss_limit_pct"])
	ddLimit := asFloat(status["max_drawdown_limit_pct"])
	if equityOK && equity > 0 && dayStart > 0 && dayLimit > 0 {
		dayLoss := (dayStart - equity) / dayStart * 100
		if dayLoss < 0 {
			dayLoss = 0
		}
		parts = append(parts, fmt.Sprintf("day loss %.1f%%/%.0f%%", dayLoss, dayLimit))
	}
	if equityOK && equity > 0 && peak > 0 && ddLimit > 0 {
		dd := (peak - equity) / peak * 100
		if dd < 0 {
			dd = 0
		}
		parts = append(parts, fmt.Sprintf("drawdown %.1f%%/%.0f%%", dd, ddLimit))
	}

	paused, _ := status["paused"].(bool)
	pauseReason := strings.TrimSpace(asString(status["pause_reason"]))
	if paused || pauseReason != "" {
		reason := pauseReason
		if reason == "" {
			reason = "paused"
		}
		parts = append(parts, "⏸ "+truncateRunes(reason, 80))
	}

	safeMode, _ := status["safe_mode"].(bool)
	safeReason := strings.TrimSpace(asString(status["safe_mode_reason"]))
	// Hard gates also set safe_mode; prefer pause_reason when already paused.
	if safeMode && !paused {
		msg := "🛡 Safe mode"
		if safeReason != "" {
			msg += " (" + truncateRunes(safeReason, 60) + ")"
		}
		parts = append(parts, msg)
	}

	blocked, _ := status["trading_blocked"].(bool)
	if blocked {
		errMsg := strings.TrimSpace(asString(status["trading_error"]))
		if errMsg != "" {
			parts = append(parts, "⚠ trading blocked: "+truncateRunes(errMsg, 60))
		} else {
			parts = append(parts, "⚠ trading blocked")
		}
	}

	wallet := strings.ToLower(strings.TrimSpace(asString(status["ai_wallet_status"])))
	switch wallet {
	case "low", "empty", "error", "unavailable":
		parts = append(parts, "AI wallet "+wallet)
	}

	return strings.Join(parts, " · ")
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func asFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
