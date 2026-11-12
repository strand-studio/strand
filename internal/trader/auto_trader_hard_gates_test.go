package trader

import (
	"testing"
	"time"

	"strand/internal/store"
)

func TestDailyLossAndDrawdownPct(t *testing.T) {
	if got := dailyLossPct(100, 90); got < 9.9 || got > 10.1 {
		t.Fatalf("daily loss want ~10, got %v", got)
	}
	if got := drawdownPct(200, 150); got < 24.9 || got > 25.1 {
		t.Fatalf("drawdown want ~25, got %v", got)
	}
	if dailyLossPct(0, 50) != 0 || drawdownPct(0, 50) != 0 {
		t.Fatal("zero baseline should yield 0")
	}
}

func TestEnforceAccountRiskGatesDailyLoss(t *testing.T) {
	at := &AutoTrader{
		name:           "t",
		dayStartEquity: 100,
		peakEquity:     120,
		lastResetTime:  time.Now(),
		lastAlerts:     make(map[string]time.Time),
		config: AutoTraderConfig{
			StrategyConfig: &store.StrategyConfig{
				RiskControl: store.RiskControlConfig{
					DailyLossLimitPct: 10,
					MaxDrawdownPct:    50, // high so daily loss trips first
				},
			},
		},
	}
	blocked, reason := at.enforceAccountRiskGates(89)
	if !blocked {
		t.Fatalf("expected daily loss block, reason=%q", reason)
	}
	if !time.Now().Before(at.stopUntil) {
		t.Fatal("expected stopUntil in the future")
	}
}

func TestEnforceAccountRiskGatesDrawdown(t *testing.T) {
	at := &AutoTrader{
		name:           "t",
		dayStartEquity: 100,
		peakEquity:     200,
		lastResetTime:  time.Now(),
		lastAlerts:     make(map[string]time.Time),
		config: AutoTraderConfig{
			StrategyConfig: &store.StrategyConfig{
				RiskControl: store.RiskControlConfig{
					DailyLossLimitPct: 0,
					MaxDrawdownPct:    25,
				},
			},
		},
	}
	blocked, reason := at.enforceAccountRiskGates(140) // 30% from peak 200
	if !blocked {
		t.Fatalf("expected drawdown block, reason=%q", reason)
	}
}

func TestPauseResumeTrading(t *testing.T) {
	at := &AutoTrader{
		name:       "t",
		lastAlerts: make(map[string]time.Time),
	}
	until := at.PauseTrading(time.Hour, "manual")
	if until.IsZero() || !time.Now().Before(at.stopUntil) {
		t.Fatal("pause did not set stopUntil")
	}
	if !at.isSafeMode() {
		t.Fatal("expected safe mode while paused")
	}
	at.ResumeTrading()
	if time.Now().Before(at.stopUntil) {
		t.Fatal("resume should clear stopUntil")
	}
	if at.isSafeMode() {
		t.Fatal("resume should clear safe mode")
	}
}
