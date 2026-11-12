package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Service) resolveTraderID(c *gin.Context) string {
	// Prefer ownership-checked resolution for console APIs.
	if id := s.resolveOwnedTraderID(c); id != "" {
		return id
	}
	st := s.eng.Store()
	userID := currentUserID(c)
	if st == nil || userID == "" {
		return ""
	}
	list, err := st.Trader().List(userID)
	if err != nil || len(list) == 0 {
		return ""
	}
	return list[0].ID
}

func (s *Service) consoleTraderStatus(c *gin.Context) {
	id := s.resolveTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	status, err := s.GetTrader(c.Request.Context(), id)
	if err != nil {
		// Fall back to store row when runtime instance is not loaded yet.
		st := s.eng.Store()
		if st != nil {
			if tr, e := st.Trader().GetByID(id); e == nil && tr != nil {
				writeJSON(c, http.StatusOK, gin.H{
					"trader_id":       tr.ID,
					"trader_name":     tr.Name,
					"ai_model":        tr.AIModelID,
					"is_running":      tr.IsRunning,
					"start_time":      tr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
					"runtime_minutes": 0,
					"call_count":      0,
					"initial_balance": tr.InitialBalance,
					"scan_interval":   "",
					"stop_until":      "",
					"last_reset_time": "",
					"ai_provider":     tr.AIModelID,
					"exchange":        tr.ExchangeID,
				})
				return
			}
		}
		writeErr(c, http.StatusNotFound, err)
		return
	}
	writeJSON(c, http.StatusOK, status)
}

func (s *Service) consoleAccount(c *gin.Context) {
	id := s.resolveTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	tm := s.eng.Traders()
	if tm == nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	at, err := tm.GetTrader(id)
	if err != nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	info, err := at.GetAccountInfo()
	if err != nil {
		writeJSON(c, http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	writeJSON(c, http.StatusOK, info)
}

func (s *Service) consolePositions(c *gin.Context) {
	id := s.resolveTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	tm := s.eng.Traders()
	if tm == nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	at, err := tm.GetTrader(id)
	if err != nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	positions, err := at.GetPositions()
	if err != nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	if positions == nil {
		positions = []map[string]any{}
	}
	writeJSON(c, http.StatusOK, positions)
}

func (s *Service) consoleDecisions(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	id := s.resolveTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	records, err := st.Decision().GetLatestRecords(id, limit)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	for _, rec := range records {
		enrichDecisionFromEquity(st, rec)
	}
	writeJSON(c, http.StatusOK, records)
}

func (s *Service) consoleDecisionsLatest(c *gin.Context) {
	s.consoleDecisions(c)
}

func (s *Service) consoleStatistics(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	id := s.resolveTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	stats, err := st.Decision().GetStatistics(id)
	if err != nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	writeJSON(c, http.StatusOK, stats)
}

func (s *Service) consoleStatisticsFull(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	id := s.resolveTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	initial := 0.0
	if tr, err := st.Trader().GetByID(id); err == nil && tr != nil {
		initial = tr.InitialBalance
	}
	stats, err := st.Position().GetFullStats(id, initial)
	if err != nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	writeJSON(c, http.StatusOK, stats)
}

func (s *Service) consoleCompetition(c *gin.Context) {
	tm := s.eng.Traders()
	if tm == nil {
		writeJSON(c, http.StatusOK, gin.H{"traders": []any{}, "count": 0, "rules": s.competitionRulesJSON()})
		return
	}
	data, err := tm.GetCompetitionData()
	if err != nil {
		writeJSON(c, http.StatusOK, gin.H{"traders": []any{}, "count": 0, "rules": s.competitionRulesJSON()})
		return
	}
	rules := s.competitionRulesJSON()
	data["rules"] = rules
	scoring, _ := rules["scoring"].(string)
	traders, _ := data["traders"].([]map[string]interface{})
	if len(traders) == 0 {
		writeJSON(c, http.StatusOK, data)
		return
	}

	switch scoring {
	case "normalized_initial":
		ref, _ := rules["reference_initial_usd"].(float64)
		if ref <= 0 {
			ref = 1000
		}
		for _, t := range traders {
			pnl, _ := toFloat(t["total_pnl"])
			t["score_pct"] = round2(pnl / ref * 100)
			t["scoring"] = "normalized_initial"
		}
	case "window_equity":
		s.applyWindowEquityScores(traders, rules)
	default:
		// Keep default total_pnl_pct ranking from GetCompetitionData.
		writeJSON(c, http.StatusOK, data)
		return
	}

	sort.Slice(traders, func(i, j int) bool {
		a, _ := toFloat(traders[i]["score_pct"])
		b, _ := toFloat(traders[j]["score_pct"])
		return a > b
	})
	data["traders"] = traders
	writeJSON(c, http.StatusOK, data)
}

// applyWindowEquityScores ranks traders by equity change inside [start_at, end_at]
// using stored equity snapshots (falls back to lifetime PnL % when snapshots missing).
func (s *Service) applyWindowEquityScores(traders []map[string]interface{}, rules gin.H) {
	st := s.eng.Store()
	startAt, _ := rules["start_at"].(string)
	endAt, _ := rules["end_at"].(string)
	var start, end time.Time
	var err error
	if strings.TrimSpace(startAt) != "" {
		start, err = time.Parse(time.RFC3339, startAt)
		if err != nil {
			start = time.Time{}
		}
	}
	if strings.TrimSpace(endAt) != "" {
		end, err = time.Parse(time.RFC3339, endAt)
		if err != nil {
			end = time.Time{}
		}
	}
	if start.IsZero() && end.IsZero() {
		// Default window: last 7 days.
		end = time.Now().UTC()
		start = end.Add(-7 * 24 * time.Hour)
	} else if end.IsZero() {
		end = time.Now().UTC()
	} else if start.IsZero() {
		start = end.Add(-7 * 24 * time.Hour)
	}

	for _, t := range traders {
		t["scoring"] = "window_equity"
		t["window_start"] = start.UTC().Format(time.RFC3339)
		t["window_end"] = end.UTC().Format(time.RFC3339)
		id, _ := t["trader_id"].(string)
		if id == "" {
			id, _ = t["id"].(string)
		}
		fallback, _ := toFloat(t["total_pnl_pct"])
		t["score_pct"] = round2(fallback)
		if st == nil || id == "" {
			t["score_source"] = "lifetime_pnl_pct"
			continue
		}
		snaps, err := st.Equity().GetByTimeRange(id, start.UTC(), end.UTC())
		if err != nil || len(snaps) < 2 {
			t["score_source"] = "lifetime_pnl_pct"
			continue
		}
		first := snaps[0].TotalEquity
		last := snaps[len(snaps)-1].TotalEquity
		if first <= 0 {
			t["score_source"] = "lifetime_pnl_pct"
			continue
		}
		t["score_pct"] = round2((last - first) / first * 100)
		t["window_start_equity"] = round2(first)
		t["window_end_equity"] = round2(last)
		t["score_source"] = "equity_snapshots"
	}
}

func (s *Service) competitionRulesJSON() gin.H {
	out := gin.H{
		"reference_initial_usd": 1000.0,
		"scoring":               "pnl_pct",
		"start_at":              nil,
		"end_at":                nil,
		"description":           "Ranked by total PnL %. Use scoring=normalized_initial or window_equity (equity snapshots in start_at..end_at).",
		"scoring_options":       []string{"pnl_pct", "normalized_initial", "window_equity"},
	}
	st := s.eng.Store()
	if st == nil {
		return out
	}
	cfg, err := st.CompetitionSettings().Get()
	if err != nil || cfg == nil {
		return out
	}
	out["reference_initial_usd"] = cfg.ReferenceInitialUSD
	out["scoring"] = cfg.Scoring
	if cfg.StartAt != nil {
		out["start_at"] = cfg.StartAt.UTC().Format(time.RFC3339)
	}
	if cfg.EndAt != nil {
		out["end_at"] = cfg.EndAt.UTC().Format(time.RFC3339)
	}
	out["description"] = "Fair-play rules: reference initial $" +
		strconv.FormatFloat(cfg.ReferenceInitialUSD, 'f', 0, 64) +
		", scoring=" + cfg.Scoring
	return out
}

func (s *Service) consoleCompetitionRulesGet(c *gin.Context) {
	writeJSON(c, http.StatusOK, s.competitionRulesJSON())
}

func (s *Service) consoleCompetitionRulesPut(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		ReferenceInitialUSD float64 `json:"reference_initial_usd"`
		Scoring             string  `json:"scoring"`
		StartAt             string  `json:"start_at"`
		EndAt               string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	var startAt, endAt *time.Time
	if strings.TrimSpace(body.StartAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartAt))
		if err != nil {
			writeJSON(c, http.StatusBadRequest, gin.H{"error": "start_at must be RFC3339"})
			return
		}
		startAt = &t
	}
	if strings.TrimSpace(body.EndAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(body.EndAt))
		if err != nil {
			writeJSON(c, http.StatusBadRequest, gin.H{"error": "end_at must be RFC3339"})
			return
		}
		endAt = &t
	}
	cfg, err := st.CompetitionSettings().Update(body.ReferenceInitialUSD, body.Scoring, startAt, endAt)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	// Bust competition cache by zeroing timestamp via a fresh fetch next time.
	if tm := s.eng.Traders(); tm != nil {
		tm.InvalidateCompetitionCache()
	}
	writeJSON(c, http.StatusOK, gin.H{
		"ok":    true,
		"rules": cfg,
	})
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
