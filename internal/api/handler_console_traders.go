package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"strand/internal/logger"
	"strand/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Service) traderInfoFromStore(tr *store.Trader, st *store.Store) gin.H {
	aiName := tr.AIModelID
	strategyName := ""
	if st != nil {
		if m, err := st.AIModel().Get(tr.UserID, tr.AIModelID); err == nil && m != nil {
			aiName = m.Name
			if aiName == "" {
				aiName = m.Provider
			}
		}
		if tr.StrategyID != "" {
			if strat, err := st.Strategy().Get(tr.UserID, tr.StrategyID); err == nil && strat != nil {
				strategyName = strat.Name
			}
		}
	}
	isRunning := tr.IsRunning
	var lastErr, lastErrAt, startupWarning string
	var scanInterval string
	if tm := s.eng.Traders(); tm != nil {
		if loadErr := tm.GetLoadError(tr.ID); loadErr != nil {
			startupWarning = loadErr.Error()
			lastErr = loadErr.Error()
		}
		if at, err := tm.GetTrader(tr.ID); err == nil && at != nil {
			stMap := at.GetStatus()
			if v, ok := stMap["is_running"].(bool); ok {
				isRunning = v
			}
			if v, ok := stMap["ai_model"].(string); ok && v != "" {
				aiName = v
			}
			if v, ok := stMap["last_runtime_error"].(string); ok && v != "" {
				lastErr = v
			}
			if v, ok := stMap["last_runtime_error_at"].(string); ok && v != "" {
				lastErrAt = v
			}
			if v, ok := stMap["scan_interval"].(string); ok && v != "" {
				scanInterval = v
			}
		}
	}
	out := gin.H{
		"trader_id":              tr.ID,
		"trader_name":            tr.Name,
		"ai_model":               aiName,
		"exchange_id":            tr.ExchangeID,
		"is_running":             isRunning,
		"show_in_competition":    tr.ShowInCompetition,
		"strategy_id":            tr.StrategyID,
		"strategy_name":          strategyName,
		"custom_prompt":          tr.CustomPrompt,
		"use_ai500":              tr.UseAI500,
		"use_oi_top":             tr.UseOITop,
		"system_prompt_template": tr.SystemPromptTemplate,
		"scan_interval_minutes":  tr.ScanIntervalMinutes,
	}
	if scanInterval != "" {
		out["scan_interval"] = scanInterval
	}
	if lastErr != "" {
		out["last_runtime_error"] = lastErr
	}
	if lastErrAt != "" {
		out["last_runtime_error_at"] = lastErrAt
	}
	if startupWarning != "" {
		out["startup_warning"] = startupWarning
	}
	return out
}

func (s *Service) consoleMyTraders(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	list, err := st.Trader().List(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, tr := range list {
		out = append(out, s.traderInfoFromStore(tr, st))
	}
	writeJSON(c, http.StatusOK, out)
}

func (s *Service) consolePublicTraders(c *gin.Context) {
	// Same shape as my-traders for now (single-tenant self-host).
	s.consoleMyTraders(c)
}

type createTraderBody struct {
	Name                string `json:"name"`
	AIModelID           string `json:"ai_model_id"`
	ExchangeID          string `json:"exchange_id"`
	StrategyID          string `json:"strategy_id"`
	ScanIntervalMinutes int    `json:"scan_interval_minutes"`
	IsCrossMargin       *bool  `json:"is_cross_margin"`
	ShowInCompetition   *bool  `json:"show_in_competition"`
	BTCETHLeverage      int    `json:"btc_eth_leverage"`
	AltcoinLeverage     int    `json:"altcoin_leverage"`
	TradingSymbols      string `json:"trading_symbols"`
	CustomPrompt        string `json:"custom_prompt"`
	OverrideBasePrompt  bool   `json:"override_base_prompt"`
	SystemPromptTemplate string `json:"system_prompt_template"`
	UseAI500            bool   `json:"use_ai500"`
	UseOITop            bool   `json:"use_oi_top"`
}

func (s *Service) consoleCreateTrader(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	var body createTraderBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || body.AIModelID == "" || body.ExchangeID == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "name / ai_model_id / exchange_id 必填"})
		return
	}
	scan := body.ScanIntervalMinutes
	if scan <= 0 {
		scan = 5
	}
	cross := true
	if body.IsCrossMargin != nil {
		cross = *body.IsCrossMargin
	}
	show := true
	if body.ShowInCompetition != nil {
		show = *body.ShowInCompetition
	}
	btcLev := body.BTCETHLeverage
	if btcLev <= 0 {
		btcLev = 5
	}
	altLev := body.AltcoinLeverage
	if altLev <= 0 {
		altLev = 5
	}
	tr := &store.Trader{
		ID:                   uuid.NewString(),
		UserID:               userID,
		Name:                 name,
		AIModelID:            body.AIModelID,
		ExchangeID:           body.ExchangeID,
		StrategyID:           body.StrategyID,
		InitialBalance:       0,
		ScanIntervalMinutes:  scan,
		IsRunning:            false,
		IsCrossMargin:        cross,
		ShowInCompetition:    show,
		BTCETHLeverage:       btcLev,
		AltcoinLeverage:      altLev,
		TradingSymbols:       body.TradingSymbols,
		UseAI500:             body.UseAI500,
		UseOITop:             body.UseOITop,
		CustomPrompt:         body.CustomPrompt,
		OverrideBasePrompt:   body.OverrideBasePrompt,
		SystemPromptTemplate: body.SystemPromptTemplate,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	if err := st.Trader().Create(tr); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	if tm := s.eng.Traders(); tm != nil {
		if err := tm.LoadUserTradersFromStore(st, userID); err != nil {
			logger.Warnf("reload traders after create: %v", err)
		}
	}
	writeJSON(c, http.StatusOK, s.traderInfoFromStore(tr, st))
}

func (s *Service) consoleUpdateTrader(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	tr, err := st.Trader().GetByID(id)
	if err != nil || tr == nil || tr.UserID != userID {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	var body createTraderBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if name := strings.TrimSpace(body.Name); name != "" {
		tr.Name = name
	}
	if body.AIModelID != "" {
		tr.AIModelID = body.AIModelID
	}
	if body.ExchangeID != "" {
		tr.ExchangeID = body.ExchangeID
	}
	if body.StrategyID != "" {
		tr.StrategyID = body.StrategyID
	}
	if body.ScanIntervalMinutes > 0 {
		tr.ScanIntervalMinutes = body.ScanIntervalMinutes
	}
	if body.IsCrossMargin != nil {
		tr.IsCrossMargin = *body.IsCrossMargin
	}
	if body.ShowInCompetition != nil {
		tr.ShowInCompetition = *body.ShowInCompetition
	}
	if body.BTCETHLeverage > 0 {
		tr.BTCETHLeverage = body.BTCETHLeverage
	}
	if body.AltcoinLeverage > 0 {
		tr.AltcoinLeverage = body.AltcoinLeverage
	}
	tr.TradingSymbols = body.TradingSymbols
	tr.CustomPrompt = body.CustomPrompt
	tr.OverrideBasePrompt = body.OverrideBasePrompt
	tr.SystemPromptTemplate = body.SystemPromptTemplate
	tr.UseAI500 = body.UseAI500
	tr.UseOITop = body.UseOITop
	tr.UpdatedAt = time.Now().UTC()

	// Detect structural changes that require a full reload vs hot path
	// (scan interval and/or leverage can apply live without restart).
	prev, _ := st.Trader().GetByID(id)
	structural := prev == nil ||
		prev.AIModelID != tr.AIModelID ||
		prev.ExchangeID != tr.ExchangeID ||
		prev.StrategyID != tr.StrategyID ||
		prev.Name != tr.Name ||
		prev.IsCrossMargin != tr.IsCrossMargin ||
		prev.TradingSymbols != tr.TradingSymbols ||
		prev.CustomPrompt != tr.CustomPrompt ||
		prev.OverrideBasePrompt != tr.OverrideBasePrompt ||
		prev.SystemPromptTemplate != tr.SystemPromptTemplate ||
		prev.UseAI500 != tr.UseAI500 ||
		prev.UseOITop != tr.UseOITop
	leverageChanged := prev != nil &&
		(prev.BTCETHLeverage != tr.BTCETHLeverage || prev.AltcoinLeverage != tr.AltcoinLeverage)
	scanChanged := prev != nil && prev.ScanIntervalMinutes != tr.ScanIntervalMinutes
	hotOnly := !structural && (scanChanged || leverageChanged)

	if err := st.Trader().Update(tr); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	if tm := s.eng.Traders(); tm != nil {
		if at, err := tm.GetTrader(id); err == nil && at != nil && hotOnly {
			stMap := at.GetStatus()
			if running, _ := stMap["is_running"].(bool); running {
				if scanChanged {
					at.SetScanInterval(time.Duration(tr.ScanIntervalMinutes) * time.Minute)
				}
				if leverageChanged && strings.TrimSpace(tr.StrategyID) != "" {
					if strat, sErr := st.Strategy().Get(userID, tr.StrategyID); sErr == nil && strat != nil {
						if cfg, pErr := strat.ParseConfig(); pErr == nil && cfg != nil {
							if tr.BTCETHLeverage > 0 {
								cfg.RiskControl.BTCETHMaxLeverage = tr.BTCETHLeverage
							}
							if tr.AltcoinLeverage > 0 {
								cfg.RiskControl.AltcoinMaxLeverage = tr.AltcoinLeverage
							}
							cfg.ClampLimits()
							if raw, mErr := json.Marshal(cfg); mErr == nil {
								strat.Config = string(raw)
								strat.UpdatedAt = time.Now().UTC()
								_ = st.Strategy().Update(strat)
								_ = at.ForceReloadStrategyConfig()
							}
						}
					}
				}
				writeJSON(c, http.StatusOK, s.traderInfoFromStore(tr, st))
				return
			}
		}
		tm.RemoveTrader(id)
		_ = tm.LoadUserTradersFromStore(st, userID)
	}
	writeJSON(c, http.StatusOK, s.traderInfoFromStore(tr, st))
}

func (s *Service) consoleTraderRuntime(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	tr, err := st.Trader().GetByID(id)
	if err != nil || tr == nil || tr.UserID != userID {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "trader not found", "error_key": "trader.not_found"})
		return
	}
	var body struct {
		ScanIntervalMinutes *int `json:"scan_interval_minutes"`
		BTCETHLeverage      *int `json:"btc_eth_leverage"`
		AltcoinLeverage     *int `json:"altcoin_leverage"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if body.ScanIntervalMinutes == nil && body.BTCETHLeverage == nil && body.AltcoinLeverage == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "runtime fields required", "error_key": "trader.runtime.invalid"})
		return
	}

	scanApplied := false
	levApplied := false
	if body.ScanIntervalMinutes != nil {
		scan := *body.ScanIntervalMinutes
		if scan <= 0 {
			writeJSON(c, http.StatusBadRequest, gin.H{"error": "scan_interval_minutes required", "error_key": "trader.runtime.invalid"})
			return
		}
		if scan > 24*60 {
			scan = 24 * 60
		}
		tr.ScanIntervalMinutes = scan
		scanApplied = true
	}
	if body.BTCETHLeverage != nil && *body.BTCETHLeverage > 0 {
		tr.BTCETHLeverage = *body.BTCETHLeverage
		levApplied = true
	}
	if body.AltcoinLeverage != nil && *body.AltcoinLeverage > 0 {
		tr.AltcoinLeverage = *body.AltcoinLeverage
		levApplied = true
	}
	tr.UpdatedAt = time.Now().UTC()
	if err := st.Trader().Update(tr); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}

	// Hot-patch linked strategy risk_control so the next cycle (and ForceReload) see new leverage.
	if levApplied && strings.TrimSpace(tr.StrategyID) != "" {
		if strat, err := st.Strategy().Get(userID, tr.StrategyID); err == nil && strat != nil {
			cfg, parseErr := strat.ParseConfig()
			if parseErr == nil && cfg != nil {
				if body.BTCETHLeverage != nil && *body.BTCETHLeverage > 0 {
					cfg.RiskControl.BTCETHMaxLeverage = *body.BTCETHLeverage
				}
				if body.AltcoinLeverage != nil && *body.AltcoinLeverage > 0 {
					cfg.RiskControl.AltcoinMaxLeverage = *body.AltcoinLeverage
				}
				cfg.ClampLimits()
				if raw, marshalErr := json.Marshal(cfg); marshalErr == nil {
					strat.Config = string(raw)
					strat.UpdatedAt = time.Now().UTC()
					_ = st.Strategy().Update(strat)
				}
			}
		}
	}

	appliedLive := false
	if tm := s.eng.Traders(); tm != nil {
		if at, err := tm.GetTrader(id); err == nil && at != nil {
			if scanApplied {
				at.SetScanInterval(time.Duration(tr.ScanIntervalMinutes) * time.Minute)
				appliedLive = true
			}
			if levApplied {
				if reloadErr := at.ForceReloadStrategyConfig(); reloadErr == nil {
					appliedLive = true
				}
			}
		}
	}
	writeJSON(c, http.StatusOK, gin.H{
		"ok":                    true,
		"scan_interval_minutes": tr.ScanIntervalMinutes,
		"btc_eth_leverage":      tr.BTCETHLeverage,
		"altcoin_leverage":      tr.AltcoinLeverage,
		"applied_live":          appliedLive,
	})
}

func (s *Service) consoleDeleteTrader(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	if tm := s.eng.Traders(); tm != nil {
		if at, err := tm.GetTrader(id); err == nil && at != nil {
			at.Stop()
		}
		tm.RemoveTrader(id)
	}
	if err := st.Trader().Delete(userID, id); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleStartTrader(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	cfg, err := st.Trader().GetFullConfig(userID, id)
	if err != nil || cfg == nil || cfg.Trader == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error":     "trader not found",
			"error_key": "trader.not_found",
		})
		return
	}

	preflight := s.runLaunchPreflight(
		userID,
		cfg.Trader.AIModelID,
		cfg.Trader.ExchangeID,
		cfg.Trader.StrategyID,
	)
	if !preflight.Ready {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error":     describeLaunchPreflightFailures(preflight),
			"error_key": "trader.start.preflight_failed",
			"preflight": preflight,
		})
		return
	}

	// Ensure in-memory trader exists after create/restart.
	if tm := s.eng.Traders(); tm != nil {
		if _, err := tm.GetTrader(id); err != nil {
			if loadErr := tm.LoadUserTradersFromStore(st, userID); loadErr != nil {
				writeJSON(c, http.StatusBadRequest, gin.H{
					"error":     fmt.Sprintf("could not load trader into runtime: %v", loadErr),
					"error_key": "trader.start.load_failed",
				})
				return
			}
		}
	}

	if err := s.StartTrader(c.Request.Context(), id); err != nil {
		if errors.Is(err, errTraderAlreadyRunning) {
			writeJSON(c, http.StatusOK, gin.H{
				"ok":        true,
				"already":   true,
				"error_key": "trader.start.already_running",
				"preflight": preflight,
			})
			return
		}
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error":     err.Error(),
			"error_key": "trader.start.failed",
		})
		return
	}
	s.eng.NotifyTelegram(fmt.Sprintf(
		"▶ Autopilot started\nName: %s\nID: %s",
		cfg.Trader.Name, id,
	))
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "preflight": preflight})
}

func (s *Service) consoleStopTrader(c *gin.Context) {
	st := s.eng.Store()
	userID := currentUserID(c)
	id := c.Param("id")
	var traderName string
	if st != nil {
		cfg, err := st.Trader().GetFullConfig(userID, id)
		if err != nil {
			writeJSON(c, http.StatusBadRequest, gin.H{
				"error":     "trader not found",
				"error_key": "trader.not_found",
			})
			return
		}
		if cfg != nil && cfg.Trader != nil {
			traderName = cfg.Trader.Name
		}
	}
	if err := s.StopTrader(c.Request.Context(), id); err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error":     err.Error(),
			"error_key": "trader.stop.failed",
		})
		return
	}
	if traderName == "" {
		traderName = id
	}
	s.eng.NotifyTelegram(fmt.Sprintf(
		"⏹ Autopilot stopped\nName: %s\nID: %s",
		traderName, id,
	))
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consolePauseTrader(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	cfg, err := st.Trader().GetFullConfig(userID, id)
	if err != nil || cfg == nil || cfg.Trader == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error": "trader not found", "error_key": "trader.not_found",
		})
		return
	}
	tm := s.eng.Traders()
	if tm == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "trading disabled", "error_key": "trading.disabled"})
		return
	}
	at, err := tm.GetTrader(id)
	if err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": err.Error(), "error_key": "trader.not_loaded"})
		return
	}
	var body struct {
		Hours  float64 `json:"hours"`
		Reason string  `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	hours := body.Hours
	if hours <= 0 {
		hours = 24
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "Paused from Dashboard"
	}
	until := at.PauseTrading(time.Duration(hours*float64(time.Hour)), reason)
	writeJSON(c, http.StatusOK, gin.H{
		"ok":          true,
		"paused":      true,
		"stop_until":  until.Format(time.RFC3339),
		"pause_reason": reason,
	})
}

func (s *Service) consoleResumeTrader(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	cfg, err := st.Trader().GetFullConfig(userID, id)
	if err != nil || cfg == nil || cfg.Trader == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error": "trader not found", "error_key": "trader.not_found",
		})
		return
	}
	tm := s.eng.Traders()
	if tm == nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "trading disabled", "error_key": "trading.disabled"})
		return
	}
	at, err := tm.GetTrader(id)
	if err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": err.Error(), "error_key": "trader.not_loaded"})
		return
	}
	at.ResumeTrading()
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "paused": false})
}

func (s *Service) consoleTraderCompetition(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	var body struct {
		ShowInCompetition bool `json:"show_in_competition"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := st.Trader().UpdateShowInCompetition(userID, id, body.ShowInCompetition); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleTraderPrompt(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	var body struct {
		CustomPrompt string `json:"custom_prompt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := st.Trader().UpdateCustomPrompt(userID, id, body.CustomPrompt, false); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleTraderConfig(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	full, err := st.Trader().GetFullConfig(userID, id)
	if err != nil || full == nil || full.Trader == nil {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	aiName := full.Trader.AIModelID
	if full.AIModel != nil {
		aiName = full.AIModel.Name
		if aiName == "" {
			aiName = full.AIModel.Provider
		}
	}
	exID := full.Trader.ExchangeID
	out := gin.H{
		"trader_id":             full.Trader.ID,
		"trader_name":           full.Trader.Name,
		"ai_model":              aiName,
		"exchange_id":           exID,
		"strategy_id":           full.Trader.StrategyID,
		"is_running":            full.Trader.IsRunning,
		"scan_interval_minutes": full.Trader.ScanIntervalMinutes,
		"custom_prompt":         full.Trader.CustomPrompt,
		"show_in_competition":   full.Trader.ShowInCompetition,
		"is_cross_margin":       full.Trader.IsCrossMargin,
		"initial_balance":       full.Trader.InitialBalance,
	}
	// Surface strategy risk sizing so Dashboard lev / hard-gate HUD matches
	// what Autopilot actually uses each cycle (not stale trader-table columns).
	if full.Strategy != nil {
		out["strategy_id"] = full.Strategy.ID
		out["strategy_name"] = full.Strategy.Name
		if cfg, perr := full.Strategy.ParseConfig(); perr == nil && cfg != nil {
			rc := cfg.RiskControl
			out["btc_eth_leverage"] = rc.BTCETHMaxLeverage
			out["altcoin_leverage"] = rc.AltcoinMaxLeverage
			out["max_positions"] = rc.MaxPositions
			out["max_margin_usage"] = rc.MaxMarginUsage
			out["daily_loss_limit_pct"] = rc.DailyLossLimitPct
			out["max_drawdown_limit_pct"] = rc.MaxDrawdownPct
			out["risk_control"] = gin.H{
				"btc_eth_max_leverage":   rc.BTCETHMaxLeverage,
				"altcoin_max_leverage":   rc.AltcoinMaxLeverage,
				"max_positions":          rc.MaxPositions,
				"max_margin_usage":       rc.MaxMarginUsage,
				"daily_loss_limit_pct":   rc.DailyLossLimitPct,
				"max_drawdown_pct":       rc.MaxDrawdownPct,
			}
		}
	}
	writeJSON(c, http.StatusOK, out)
}
