package trader

import (
	"strings"
	"testing"
)

func TestFormatOpsStatusLineHardGatePause(t *testing.T) {
	line := FormatOpsStatusLine(map[string]interface{}{
		"trader_name":            "Strand Autopilot",
		"paused":                 true,
		"pause_reason":           "daily loss 12.0% >= limit 10%",
		"safe_mode":              true, // hard gate also sets this
		"safe_mode_reason":       "daily loss",
		"day_start_equity":       1000.0,
		"peak_equity":            1100.0,
		"daily_loss_limit_pct":   10.0,
		"max_drawdown_limit_pct": 25.0,
	}, 880, true)
	for _, part := range []string{"Strand Autopilot", "⏸", "daily loss", "equity $880.00"} {
		if !strings.Contains(line, part) {
			t.Fatalf("missing %q in %q", part, line)
		}
	}
	if strings.Contains(line, "🛡 Safe mode") {
		t.Fatalf("unexpected safe mode badge in hard-gate pause: %q", line)
	}
}

func TestFormatOpsStatusLineAISafeMode(t *testing.T) {
	line := FormatOpsStatusLine(map[string]interface{}{
		"trader_name":      "Bot",
		"paused":           false,
		"safe_mode":        true,
		"safe_mode_reason": "AI failed 3 consecutive times",
		"trading_blocked":  false,
		"ai_wallet_status": "low",
	}, 0, false)
	for _, part := range []string{"🛡 Safe mode", "AI failed", "AI wallet low"} {
		if !strings.Contains(line, part) {
			t.Fatalf("missing %q in %q", part, line)
		}
	}
	if strings.Contains(line, "⏸") {
		t.Fatalf("unexpected pause marker: %q", line)
	}
}

func TestFormatOpsStatusLineTradingBlocked(t *testing.T) {
	line := FormatOpsStatusLine(map[string]interface{}{
		"trader_name":     "HL Bot",
		"trading_blocked": true,
		"trading_error":   "agent unauthorized",
	}, 250, true)
	for _, part := range []string{"⚠ trading blocked", "agent unauthorized", "$250.00"} {
		if !strings.Contains(line, part) {
			t.Fatalf("missing %q in %q", part, line)
		}
	}
}
