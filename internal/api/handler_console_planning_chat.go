package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"strand/internal/market"
	"strand/internal/mcp"
	"strand/internal/planning/core"
	"strand/internal/provider/hyperliquid"
	"strand/internal/store"
	"strand/internal/trader/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type planningChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Trade planning chat: optional model hint + deterministic compile over SSE.
// Full Frontier multi-tool agent loops are not ported; Custom compile remains available.
func (s *Service) consolePlanningChat(c *gin.Context) {
	var body struct {
		ExchangeID        string            `json:"exchange_id"`
		ModelID           string            `json:"model_id"`
		Symbol            string            `json:"symbol"`
		Venue             string            `json:"venue"`
		Interval          string            `json:"interval"`
		VisibleFrom       int64             `json:"visible_from"`
		VisibleTo         int64             `json:"visible_to"`
		Message           string            `json:"message"`
		MarketGroup       string            `json:"market_group"`
		PlanID            string            `json:"plan_id"`
		ExecutionKind     string            `json:"execution_kind"`
		CompareStructures bool              `json:"compare_structures"`
		PluginWindow      string            `json:"plugin_window"`
		SessionID         string            `json:"session_id"`
		History           []planningChatMsg `json:"history"`
		Constraints       core.Constraints  `json:"constraints"`
		AutoSize          bool              `json:"auto_size"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	venue := normalizeExchangeQuery(body.Venue)
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		writeErr(c, http.StatusBadRequest, fmt.Errorf("symbol required"))
		return
	}
	switch venue {
	case "binance", "okx", "bitget", "hyperliquid":
	default:
		writeErr(c, http.StatusBadRequest, fmt.Errorf("planning chat supports binance, okx, bitget, hyperliquid; got %s", venue))
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}

	writeSSE(c, "status", gin.H{"message": "Reading current market context…"})

	userID := currentUserID(c)
	sessionID := strings.TrimSpace(body.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	assistants := make([]planningChatMsg, 0, 4)
	emitAssistant := func(message string) {
		assistants = append(assistants, planningChatMsg{Role: "assistant", Content: message})
		writeSSE(c, "assistant", gin.H{"message": message})
	}
	now := time.Now().UTC()
	constraints := body.Constraints
	if constraints.FeeBPS == "" {
		constraints.FeeBPS = "10"
	}
	if constraints.SlippageBPS == "" {
		constraints.SlippageBPS = "10"
	}
	if constraints.MinRewardRisk == "" {
		constraints.MinRewardRisk = "0"
	}
	if body.MarketGroup == "ai500" && constraints.EntryPolicy == "" {
		constraints.EntryPolicy = core.EntryAI500TrendLong
	}

	var trader types.Trader
	if body.ExchangeID != "" {
		st := s.eng.Store()
		if st == nil {
			writeSSE(c, "error", gin.H{"message": errStoreUnavailable.Error()})
			writeSSE(c, "done", gin.H{})
			return
		}
		ex, err := st.Exchange().GetByID(userID, body.ExchangeID)
		if err != nil || ex == nil {
			writeSSE(c, "error", gin.H{"message": "exchange account not found"})
			writeSSE(c, "done", gin.H{})
			return
		}
		trader, err = buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
		if err != nil {
			writeSSE(c, "error", gin.H{"message": err.Error()})
			writeSSE(c, "done", gin.H{})
			return
		}
		if body.AutoSize {
			bal, berr := trader.GetBalance()
			if berr != nil {
				writeSSE(c, "error", gin.H{"message": berr.Error()})
				writeSSE(c, "done", gin.H{})
				return
			}
			equity, _ := floatFromBalanceMap(bal, "totalEquity", "total_equity", "totalWalletBalance")
			avail, _ := floatFromBalanceMap(bal, "availableBalance", "available_balance")
			if equity <= 0 {
				writeSSE(c, "error", gin.H{"message": "account equity is zero; fund the exchange or explore without an account"})
				writeSSE(c, "done", gin.H{})
				return
			}
			if avail < 0 {
				avail = 0
			}
			constraints.Equity = formatPlanningDecimal(equity)
			constraints.SizingPolicy = core.SizingAccountMargin
			lev := constraints.EffectiveLeverage()
			risk := equity * 0.01
			if risk <= 0 {
				risk = 1
			}
			constraints.MaxRisk = formatPlanningDecimal(risk)
			maxNotional := avail * float64(lev)
			if maxNotional <= 0 {
				maxNotional = equity * 0.05
			}
			if maxNotional <= 0 {
				maxNotional = 100
			}
			constraints.MaxNotional = formatPlanningDecimal(maxNotional)
			if avail > 0 {
				constraints.AvailableMargin = formatPlanningDecimal(avail)
				constraints.PlanningOnly = false
				writeSSE(c, "planning_sizing", gin.H{
					"message": fmt.Sprintf("Sized from account equity %s with available margin %s.", constraints.Equity, constraints.AvailableMargin),
				})
			} else {
				constraints.AvailableMargin = "0"
				constraints.PlanningOnly = true
				writeSSE(c, "planning_sizing", gin.H{
					"message": "Available margin is zero; drafts are equity previews only and cannot be approved for live submission.",
				})
			}
		}
	} else {
		if constraints.Equity == "" {
			constraints.Equity = "1000"
		}
		if constraints.MaxRisk == "" {
			constraints.MaxRisk = "10"
		}
		if constraints.MaxNotional == "" {
			constraints.MaxNotional = "100"
		}
		constraints.PlanningOnly = true
		constraints.SizingPolicy = core.SizingAccountMargin
		constraints.AvailableMargin = "0"
	}

	snap, err := s.buildPlanningSnapshot(c.Request.Context(), trader, venue, symbol, now)
	if err != nil {
		writeSSE(c, "error", gin.H{"message": err.Error()})
		writeSSE(c, "done", gin.H{})
		return
	}
	writeSSE(c, "tool_start", gin.H{"id": "get_market", "name": "get_market"})
	writeSSE(c, "tool_result", gin.H{"id": "get_market", "name": "get_market", "ok": true})

	interval := strings.TrimSpace(body.Interval)
	if interval == "" {
		interval = "1h"
	}
	writeSSE(c, "tool_start", gin.H{"id": "get_candles", "name": "get_candles", "arguments": gin.H{"interval": interval}})
	candleCtx, support, resistance, candleCount := s.loadPlanningCandleContext(c.Request.Context(), venue, symbol, interval)
	writeSSE(c, "tool_result", gin.H{
		"id": "get_candles", "name": "get_candles", "ok": candleCount > 0,
		"interval": interval, "count": candleCount,
	})

	marketCtx := candleCtx
	if extra := planningPublicMarketContext(venue, symbol); extra != "" {
		marketCtx = strings.TrimSpace(marketCtx + "\n" + extra)
		writeSSE(c, "plugin_data", gin.H{
			"plugin": "funding",
			"metrics": []gin.H{
				{"label": "Market context", "value": "1", "unit": "note"},
			},
		})
	}

	side := inferPlanningSide(body.Message, constraints.EntryPolicy)
	hint := planningAIHint{Side: side, Rationale: strings.TrimSpace(body.Message)}
	if model, merr := s.resolveChatModel(userID, body.ModelID); merr == nil && model != nil {
		writeSSE(c, "status", gin.H{"message": "Asking the selected model for a planning hint…"})
		writeSSE(c, "model_transport", gin.H{"transport": "text_tools", "model_id": model.ID, "provider": model.Provider})
		client, cerr := s.buildChatClient(model)
		if cerr != nil {
			emitAssistant("Model client unavailable (" + cerr.Error() + "). Falling back to deterministic book drafts.")
		} else if aiHint, aerr := s.askPlanningHint(c.Request.Context(), client, venue, symbol, snap, body.Message, constraints.EntryPolicy, marketCtx); aerr != nil {
			emitAssistant("Model hint unavailable (" + aerr.Error() + "). Falling back to deterministic book drafts.")
		} else {
			hint = aiHint
			if constraints.EntryPolicy == core.EntryAI500TrendLong {
				hint.Side = "long"
			}
			side = hint.Side
			emitAssistant("Model hint: " + side + ". " + strings.TrimSpace(hint.Rationale))
		}
	} else {
		emitAssistant("No model bound for this session; building deterministic drafts from the live book. Review and approve before any order is sent.")
	}

	writeSSE(c, "tool_start", gin.H{"id": "get_entry_plugins", "name": "get_entry_plugins"})
	entryPlugins := builtinEntryPlugins(sessionID)
	writeSSE(c, "tool_result", gin.H{
		"id": "get_entry_plugins", "name": "get_entry_plugins", "ok": true, "count": len(entryPlugins),
	})
	if body.AutoSize {
		equity, _ := strconv.ParseFloat(string(constraints.Equity), 64)
		bid, _ := strconv.ParseFloat(string(snap.Bid), 64)
		ask, _ := strconv.ParseFloat(string(snap.Ask), 64)
		entryPx := 0.0
		switch {
		case bid > 0 && ask > 0:
			entryPx = (bid + ask) / 2
		case ask > 0:
			entryPx = ask
		case bid > 0:
			entryPx = bid
		}
		stopPx := entryPx
		if entryPx > 0 {
			if side == "short" {
				stopPx = entryPx * 1.01
			} else {
				stopPx = entryPx * 0.99
			}
		}
		writeSSE(c, "tool_start", gin.H{
			"id": "run_entry_plugin", "name": "run_entry_plugin",
			"arguments": gin.H{"plugin_id": "risk-size-v1"},
		})
		if sized, serr := runBuiltinEntryPlugin("risk-size-v1", entryPluginInput{
			Equity:   equity,
			RiskPct:  1,
			Entry:    entryPx,
			Stop:     stopPx,
			Leverage: float64(constraints.EffectiveLeverage()),
		}); serr != nil {
			writeSSE(c, "tool_result", gin.H{
				"id": "run_entry_plugin", "name": "run_entry_plugin", "ok": false, "error": serr.Error(),
			})
		} else {
			writeSSE(c, "tool_result", gin.H{
				"id": "run_entry_plugin", "name": "run_entry_plugin", "ok": true,
				"plugin_id": "risk-size-v1", "result": sized,
			})
		}
	}

	kinds := planningChatKinds(body.ExecutionKind, body.CompareStructures)
	writeSSE(c, "tool_start", gin.H{"id": "propose_plans", "name": "propose_plans", "arguments": gin.H{"kinds": kinds, "side": side}})

	plans := make([]core.Plan, 0, len(kinds))
	checks := make([]gin.H, 0, len(kinds))
	for _, kind := range kinds {
		idea, ierr := buildPlanningChatIdea(snap, kind, hint)
		if ierr != nil {
			checks = append(checks, gin.H{
				"id": kind, "kind": kind, "name": kind, "status": "rejected", "reason": ierr.Error(),
			})
			writeSSE(c, "candidate_result", gin.H{
				"id": kind, "kind": kind, "name": kind, "status": "rejected", "reason": ierr.Error(),
			})
			continue
		}
		plan, cerr := core.Compile(snap, constraints, idea, now)
		if cerr != nil {
			checks = append(checks, gin.H{
				"id": kind, "kind": kind, "name": kind, "status": "rejected", "reason": cerr.Error(),
			})
			writeSSE(c, "candidate_result", gin.H{
				"id": kind, "kind": kind, "name": kind, "status": "rejected", "reason": cerr.Error(),
			})
			continue
		}
		st := s.userPlanningState(userID)
		planningStateMu.Lock()
		st.Records[plan.ID] = planningRecord{
			Plan:       plan,
			State:      "draft",
			ExchangeID: strings.TrimSpace(body.ExchangeID),
		}
		if body.PlanID != "" && body.PlanID != plan.ID {
			delete(st.Records, body.PlanID)
		}
		planningStateMu.Unlock()
		plans = append(plans, plan)
		checks = append(checks, gin.H{
			"id": kind, "kind": kind, "name": kind, "status": "saved", "plan_id": plan.ID,
		})
		writeSSE(c, "candidate_result", gin.H{
			"id": kind, "kind": kind, "name": kind, "status": "saved", "plan_id": plan.ID,
		})
	}
	writeSSE(c, "tool_result", gin.H{
		"id": "propose_plans", "name": "propose_plans", "ok": len(plans) > 0,
		"count": len(plans),
	})

	writeSSE(c, "plans", gin.H{
		"plans":    plans,
		"checks":   checks,
		"waiting":  false,
		"rejected": []any{},
	})

	if len(plans) > 0 {
		writeSSE(c, "tool_start", gin.H{"id": "draw_chart", "name": "draw_chart"})
		drawings := buildPlanningDrawings(plans[0], support, resistance, now)
		writeSSE(c, "drawings", gin.H{"drawings": drawings})
		writeSSE(c, "tool_result", gin.H{"id": "draw_chart", "name": "draw_chart", "ok": len(drawings) > 0, "count": len(drawings)})
	}

	summary := fmt.Sprintf("Prepared %d draft plan(s) for %s %s. Review structures, then approve to place live orders.", len(plans), side, symbol)
	if len(plans) == 0 {
		summary = "No draft passed venue and risk checks. Adjust constraints or try Custom market / limit / ladder / TWAP."
	}
	emitAssistant(summary)
	writeSSE(c, "outcome", gin.H{
		"drafts":      len(plans),
		"waiting":     false,
		"stop_reason": "completed",
	})
	s.persistPlanningConversation(userID, sessionID, body.ExchangeID, body.ModelID, symbol, body.History, body.Message, assistants)
	writeSSE(c, "done", gin.H{"session_id": sessionID})
}

func (s *Service) persistPlanningConversation(
	userID, sessionID, exchangeID, modelID, symbol string,
	history []planningChatMsg,
	userMessage string,
	assistants []planningChatMsg,
) {
	st := s.eng.Store()
	if st == nil || strings.TrimSpace(userID) == "" || strings.TrimSpace(sessionID) == "" {
		return
	}
	msgs := make([]planningChatMsg, 0, len(history)+1+len(assistants))
	for _, m := range history {
		role := strings.TrimSpace(m.Role)
		content := strings.TrimSpace(m.Content)
		if role == "" || content == "" || role == "error" {
			continue
		}
		msgs = append(msgs, planningChatMsg{Role: role, Content: content})
	}
	if text := strings.TrimSpace(userMessage); text != "" {
		msgs = append(msgs, planningChatMsg{Role: "user", Content: text})
	}
	msgs = append(msgs, assistants...)
	if len(msgs) == 0 {
		return
	}
	raw, err := json.Marshal(msgs)
	if err != nil {
		return
	}
	_ = st.PlanningConversations().Upsert(&store.PlanningConversation{
		ID:         sessionID,
		UserID:     userID,
		Symbol:     symbol,
		ExchangeID: strings.TrimSpace(exchangeID),
		ModelID:    strings.TrimSpace(modelID),
		Messages:   string(raw),
	})
}

type planningAIHint struct {
	Side         string  `json:"side"`
	Rationale    string  `json:"rationale"`
	StopFactor   float64 `json:"stop_factor"`
	TargetFactor float64 `json:"target_factor"`
}

var planningJSONBlock = regexp.MustCompile(`(?s)\{.*\}`)

func (s *Service) askPlanningHint(parent context.Context, client mcp.AIClient, venue, symbol string, snap core.Snapshot, message, entryPolicy, marketCtx string) (planningAIHint, error) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	policy := "long or short"
	if entryPolicy == core.EntryAI500TrendLong {
		policy = "long only"
	}
	prompt := fmt.Sprintf(`You are helping draft a crypto perpetual trade plan for review only. Orders are never sent from this step.
Return ONLY compact JSON:
{"side":"long"|"short","rationale":"one sentence","stop_factor":number,"target_factor":number}
Rules:
- side must be %s
- stop_factor and target_factor are multipliers of the reference price (long example stop_factor 0.97 target_factor 1.05; short example stop_factor 1.03 target_factor 0.95)
- keep |1-factor| between 0.005 and 0.08
Venue %s symbol %s bid %s ask %s
Market context:
%s
User request: %s`, policy, venue, symbol, snap.Bid, snap.Ask, marketCtx, message)
	text, err := client.CallWithRequest(&mcp.Request{
		Messages: []mcp.Message{
			mcp.NewSystemMessage("Reply with JSON only. No markdown."),
			mcp.NewUserMessage(prompt),
		},
		Ctx: ctx,
	})
	if err != nil {
		return planningAIHint{}, err
	}
	raw := strings.TrimSpace(text)
	if m := planningJSONBlock.FindString(raw); m != "" {
		raw = m
	}
	var hint planningAIHint
	if err := json.Unmarshal([]byte(raw), &hint); err != nil {
		return planningAIHint{}, fmt.Errorf("model returned non-JSON hint")
	}
	hint.Side = strings.ToLower(strings.TrimSpace(hint.Side))
	if hint.Side != "long" && hint.Side != "short" {
		return planningAIHint{}, fmt.Errorf("invalid side in model hint")
	}
	if hint.StopFactor == 0 || hint.TargetFactor == 0 {
		if hint.Side == "long" {
			hint.StopFactor, hint.TargetFactor = 0.97, 1.05
		} else {
			hint.StopFactor, hint.TargetFactor = 1.03, 0.95
		}
	}
	if hint.Rationale == "" {
		hint.Rationale = strings.TrimSpace(message)
	}
	return hint, nil
}

func inferPlanningSide(message, entryPolicy string) string {
	if entryPolicy == core.EntryAI500TrendLong {
		return "long"
	}
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "short"), strings.Contains(m, "做空"), strings.Contains(m, "空头"):
		return "short"
	default:
		return "long"
	}
}

func planningChatKinds(executionKind string, compare bool) []string {
	kind := strings.ToLower(strings.TrimSpace(executionKind))
	if compare || kind == "" || kind == "auto" {
		return []string{"market", "limit", "ladder", "twap"}
	}
	switch kind {
	case "market", "limit", "ladder", "twap":
		return []string{kind}
	default:
		return []string{"limit"}
	}
}

func buildPlanningChatIdea(snap core.Snapshot, kind string, hint planningAIHint) (core.Idea, error) {
	side := hint.Side
	if side != "long" && side != "short" {
		side = "long"
	}
	ref := snap.Ask
	if side == "short" {
		ref = snap.Bid
	}
	stopFactor, targetFactor := hint.StopFactor, hint.TargetFactor
	if stopFactor == 0 || targetFactor == 0 {
		if side == "long" {
			stopFactor, targetFactor = 0.97, 1.05
		} else {
			stopFactor, targetFactor = 1.03, 0.95
		}
	}
	// Clamp extreme model factors.
	if side == "long" {
		if stopFactor >= 1 || stopFactor < 0.92 {
			stopFactor = 0.97
		}
		if targetFactor <= 1 || targetFactor > 1.08 {
			targetFactor = 1.05
		}
	} else {
		if stopFactor <= 1 || stopFactor > 1.08 {
			stopFactor = 1.03
		}
		if targetFactor >= 1 || targetFactor < 0.92 {
			targetFactor = 0.95
		}
	}
	entry := core.Price(ref, 1, snap.Tick)
	stop := core.Price(ref, stopFactor, snap.Tick)
	target := core.Price(ref, targetFactor, snap.Tick)
	rationale := strings.TrimSpace(hint.Rationale)
	if rationale == "" {
		rationale = "Draft from the current order book"
	}
	idea := core.Idea{
		Name:      capitalizePlanningKind(kind) + " " + side,
		Side:      side,
		Entry:     entry,
		Stop:      stop,
		Target:    target,
		Rationale: rationale,
	}
	switch kind {
	case "market":
		idea.Execution = &core.Execution{Kind: "market"}
	case "limit":
		idea.Execution = &core.Execution{Kind: "limit"}
	case "ladder":
		var levels []core.Level
		if side == "long" {
			levels = []core.Level{
				{Price: core.Price(ref, 0.998, snap.Tick), Weight: "0.4"},
				{Price: core.Price(ref, 0.994, snap.Tick), Weight: "0.35"},
				{Price: core.Price(ref, 0.99, snap.Tick), Weight: "0.25"},
			}
		} else {
			levels = []core.Level{
				{Price: core.Price(ref, 1.002, snap.Tick), Weight: "0.4"},
				{Price: core.Price(ref, 1.006, snap.Tick), Weight: "0.35"},
				{Price: core.Price(ref, 1.01, snap.Tick), Weight: "0.25"},
			}
		}
		idea.Entry = levels[0].Price
		idea.Execution = &core.Execution{Kind: "ladder", Levels: levels}
	case "twap":
		idea.Execution = &core.Execution{Kind: "twap", Slices: 4, IntervalSeconds: 30}
	default:
		return core.Idea{}, fmt.Errorf("unsupported execution kind %s", kind)
	}
	return idea, nil
}

func capitalizePlanningKind(kind string) string {
	if kind == "" {
		return kind
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func planningPublicMarketContext(venue, symbol string) string {
	marketSymbol := planningPublicMarketSymbol(venue, symbol)
	data, err := market.Get(marketSymbol)
	if err != nil || data == nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("mark≈%.6g 1h=%+.2f%% 4h=%+.2f%% funding=%.2e", data.CurrentPrice, data.PriceChange1h, data.PriceChange4h, data.FundingRate),
	}
	if data.OpenInterest != nil {
		parts = append(parts, fmt.Sprintf("oi_latest=%.4g oi_avg=%.4g", data.OpenInterest.Latest, data.OpenInterest.Average))
	}
	return strings.Join(parts, " ")
}

func (s *Service) loadPlanningCandleContext(ctx context.Context, venue, symbol, interval string) (summary string, support, resistance float64, count int) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	type bar struct{ high, low, close float64 }
	bars := make([]bar, 0, 80)
	switch venue {
	case "hyperliquid", "lighter", "aster":
		client := hyperliquid.NewClient()
		rows, err := client.GetCandles(ctx, symbol, hyperliquid.MapTimeframe(interval), 64)
		if err != nil {
			return "candles unavailable", 0, 0, 0
		}
		for _, candle := range rows {
			h, _ := strconv.ParseFloat(candle.High, 64)
			l, _ := strconv.ParseFloat(candle.Low, 64)
			cl, _ := strconv.ParseFloat(candle.Close, 64)
			bars = append(bars, bar{h, l, cl})
		}
	default:
		rows, err := market.FetchCEXKlines(ctx, venue, symbol, interval, 64)
		if err != nil {
			return "candles unavailable", 0, 0, 0
		}
		for _, k := range rows {
			bars = append(bars, bar{k.High, k.Low, k.Close})
		}
	}
	if len(bars) == 0 {
		return "candles empty", 0, 0, 0
	}
	hi, lo := bars[0].high, bars[0].low
	for _, b := range bars {
		if b.high > hi {
			hi = b.high
		}
		if b.low < lo && b.low > 0 {
			lo = b.low
		}
	}
	last := bars[len(bars)-1].close
	chg := 0.0
	if bars[0].close > 0 {
		chg = (last - bars[0].close) / bars[0].close * 100
	}
	summary = fmt.Sprintf("%s candles=%d last=%.6g range=[%.6g,%.6g] change=%+.2f%%", interval, len(bars), last, lo, hi, chg)
	return summary, lo, hi, len(bars)
}

func buildPlanningDrawings(plan core.Plan, support, resistance float64, now time.Time) []gin.H {
	t := now.Unix()
	out := make([]gin.H, 0, 5)
	add := func(label, price string) {
		if !core.Positive(core.Decimal(price)) {
			return
		}
		out = append(out, gin.H{
			"kind":  "horizontal_line",
			"label": label,
			"points": []gin.H{
				{"time": t, "price": price},
			},
		})
	}
	if support > 0 && !math.IsNaN(support) {
		add("S1 support", string(formatPlanningDecimal(support)))
	}
	if resistance > 0 && !math.IsNaN(resistance) {
		add("R1 resistance", string(formatPlanningDecimal(resistance)))
	}
	add("Plan entry", string(plan.Idea.Entry))
	add("Plan stop", string(plan.Idea.Stop))
	add("Plan target", string(plan.Idea.Target))
	return out
}
