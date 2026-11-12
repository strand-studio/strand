package trader

import (
	"fmt"
	"time"
)

const (
	defaultHardGatePause = 24 * time.Hour
	alertThrottle        = 15 * time.Minute
)

// SetNotifyAlert wires a throttled alert sink (e.g. Telegram).
func (at *AutoTrader) SetNotifyAlert(fn func(string)) {
	if at == nil {
		return
	}
	at.alertMu.Lock()
	at.notifyAlert = fn
	at.alertMu.Unlock()
}

func (at *AutoTrader) notifyAlertThrottled(key, text string) {
	if at == nil || text == "" {
		return
	}
	at.alertMu.Lock()
	fn := at.notifyAlert
	last := at.lastAlerts[key]
	if time.Since(last) < alertThrottle {
		at.alertMu.Unlock()
		return
	}
	if at.lastAlerts == nil {
		at.lastAlerts = make(map[string]time.Time)
	}
	at.lastAlerts[key] = time.Now()
	at.alertMu.Unlock()
	if fn != nil {
		fn(text)
	}
}

// PauseTrading blocks new Autopilot risk-taking for duration.
// Existing positions stay open; the loop keeps running.
func (at *AutoTrader) PauseTrading(duration time.Duration, reason string) time.Time {
	if at == nil {
		return time.Time{}
	}
	if duration <= 0 {
		duration = defaultHardGatePause
	}
	until := time.Now().Add(duration)
	at.stopUntil = until
	at.pauseReason = reason
	at.runtimeHealthMu.Lock()
	at.safeMode = true
	at.safeModeReason = reason
	at.runtimeHealthMu.Unlock()
	at.notifyAlertThrottled("pause", fmt.Sprintf(
		"⏸ Autopilot paused\nName: %s\nUntil: %s\nReason: %s",
		at.name, until.Format(time.RFC3339), reason,
	))
	at.logWarnf("⏸ Trading paused until %s — %s", until.Format(time.RFC3339), reason)
	return until
}

// ResumeTrading clears a manual / hard-gate pause and safe mode.
func (at *AutoTrader) ResumeTrading() {
	if at == nil {
		return
	}
	at.stopUntil = time.Time{}
	at.pauseReason = ""
	at.runtimeHealthMu.Lock()
	at.safeMode = false
	at.safeModeReason = ""
	at.runtimeHealthMu.Unlock()
	at.notifyAlertThrottled("resume", fmt.Sprintf(
		"▶ Autopilot resumed\nName: %s", at.name,
	))
	at.logInfof("▶ Trading resume requested — hard gates cleared")
}

// enforceAccountRiskGates updates peak/day equity and pauses when strategy
// hard limits are breached. Returns true when the cycle should skip new risk.
func (at *AutoTrader) enforceAccountRiskGates(equity float64) (blocked bool, reason string) {
	if at == nil || equity <= 0 {
		return false, ""
	}

	// Calendar-day reset for daily loss tracking.
	now := time.Now()
	if now.YearDay() != at.lastResetTime.YearDay() || now.Year() != at.lastResetTime.Year() {
		at.dailyPnL = 0
		at.dayStartEquity = equity
		at.lastResetTime = now
		at.logInfof("📅 Daily P&L / day-start equity reset (%.2f)", equity)
	}
	if at.dayStartEquity <= 0 {
		at.dayStartEquity = equity
	}
	if equity > at.peakEquity {
		at.peakEquity = equity
	}
	if at.peakEquity <= 0 {
		at.peakEquity = equity
	}

	if time.Now().Before(at.stopUntil) {
		remaining := at.stopUntil.Sub(time.Now())
		reason = at.pauseReason
		if reason == "" {
			reason = fmt.Sprintf("Risk control paused, remaining %.0f minutes", remaining.Minutes())
		}
		return true, reason
	}

	risk := storeRiskControl(at)
	if risk.DailyLossLimitPct > 0 && at.dayStartEquity > 0 {
		lossPct := (at.dayStartEquity - equity) / at.dayStartEquity * 100
		if lossPct >= risk.DailyLossLimitPct {
			reason = fmt.Sprintf(
				"Daily loss %.1f%% reached limit %.1f%% (day-start %.2f → %.2f)",
				lossPct, risk.DailyLossLimitPct, at.dayStartEquity, equity,
			)
			at.PauseTrading(defaultHardGatePause, reason)
			return true, reason
		}
	}
	if risk.MaxDrawdownPct > 0 && at.peakEquity > 0 {
		ddPct := (at.peakEquity - equity) / at.peakEquity * 100
		if ddPct >= risk.MaxDrawdownPct {
			reason = fmt.Sprintf(
				"Max drawdown %.1f%% reached limit %.1f%% (peak %.2f → %.2f)",
				ddPct, risk.MaxDrawdownPct, at.peakEquity, equity,
			)
			at.PauseTrading(defaultHardGatePause, reason)
			return true, reason
		}
	}
	return false, ""
}

type riskLimits struct {
	DailyLossLimitPct float64
	MaxDrawdownPct    float64
}

func storeRiskControl(at *AutoTrader) riskLimits {
	if at == nil || at.config.StrategyConfig == nil {
		return riskLimits{}
	}
	rc := at.config.StrategyConfig.RiskControl
	return riskLimits{
		DailyLossLimitPct: rc.DailyLossLimitPct,
		MaxDrawdownPct:    rc.MaxDrawdownPct,
	}
}

// NotifyCrashed records the error for the UI and pushes a throttled alert.
func (at *AutoTrader) NotifyCrashed(err error) {
	if at == nil || err == nil {
		return
	}
	at.RecordRuntimeError(err.Error())
	at.notifyAlertThrottled("crash", fmt.Sprintf(
		"⏹ Autopilot crashed\nName: %s\nError: %v", at.name, err,
	))
}

// SetScanInterval hot-updates the cycle ticker without stopping the trader.
func (at *AutoTrader) SetScanInterval(d time.Duration) {
	if at == nil {
		return
	}
	if d < time.Minute {
		d = time.Minute
	}
	if d > 24*time.Hour {
		d = 24 * time.Hour
	}
	at.config.ScanInterval = d
	if at.resetTickerCh == nil {
		return
	}
	select {
	case at.resetTickerCh <- d:
	default:
		select {
		case <-at.resetTickerCh:
		default:
		}
		select {
		case at.resetTickerCh <- d:
		default:
		}
	}
}

// AccountLossPct helpers for unit tests.
func dailyLossPct(dayStart, equity float64) float64 {
	if dayStart <= 0 {
		return 0
	}
	return (dayStart - equity) / dayStart * 100
}

func drawdownPct(peak, equity float64) float64 {
	if peak <= 0 {
		return 0
	}
	return (peak - equity) / peak * 100
}
