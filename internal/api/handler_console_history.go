package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"strand/internal/store"

	"github.com/gin-gonic/gin"
)

func (s *Service) resolveOwnedTraderID(c *gin.Context) string {
	id := strings.TrimSpace(c.Query("trader_id"))
	st := s.eng.Store()
	userID := currentUserID(c)
	if st == nil || userID == "" {
		return ""
	}
	if id == "" {
		list, err := st.Trader().List(userID)
		if err != nil || len(list) == 0 {
			return ""
		}
		return list[0].ID
	}
	tr, err := st.Trader().GetByID(id)
	if err != nil || tr == nil || tr.UserID != userID {
		return ""
	}
	return tr.ID
}

func (s *Service) traderInitialBalance(st *store.Store, traderID string) float64 {
	if st == nil {
		return 0
	}
	if tr, err := st.Trader().GetByID(traderID); err == nil && tr != nil && tr.InitialBalance > 0 {
		return tr.InitialBalance
	}
	return 0
}

func equityPointsFromSnapshots(snaps []*store.EquitySnapshot, initialBalance float64) []gin.H {
	out := make([]gin.H, 0, len(snaps))
	baseline := initialBalance
	if baseline <= 0 && len(snaps) > 0 {
		baseline = snaps[0].TotalEquity
	}
	for i, snap := range snaps {
		if snap == nil {
			continue
		}
		pnl := 0.0
		pnlPct := 0.0
		if baseline > 0 {
			pnl = snap.TotalEquity - baseline
			pnlPct = (pnl / baseline) * 100
		}
		out = append(out, gin.H{
			"timestamp":      snap.Timestamp.UTC().Format(time.RFC3339),
			"total_equity":   snap.TotalEquity,
			"balance":        snap.Balance,
			"available":      snap.AvailableBalance,
			"unrealized_pnl": snap.UnrealizedPnL,
			"position_count": snap.PositionCount,
			"margin_used_pct": snap.MarginUsedPct,
			"pnl":            pnl,
			"pnl_pct":        pnlPct,
			"total_pnl_pct":  pnlPct,
			"cycle_number":   i + 1,
		})
	}
	return out
}

func (s *Service) consoleEquityHistory(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	id := s.resolveOwnedTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	limit := 2000
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	snaps, err := st.Equity().GetLatest(id, limit)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, equityPointsFromSnapshots(snaps, s.traderInitialBalance(st, id)))
}

func (s *Service) consoleEquityHistoryBatch(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{"histories": gin.H{}})
		return
	}
	userID := currentUserID(c)
	var body struct {
		TraderIDs []string `json:"trader_ids"`
		Hours     int      `json:"hours"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}

	histories := make(map[string][]gin.H, len(body.TraderIDs))
	now := time.Now().UTC()
	for _, rawID := range body.TraderIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		tr, err := st.Trader().GetByID(id)
		if err != nil || tr == nil || tr.UserID != userID {
			histories[id] = []gin.H{}
			continue
		}
		var snaps []*store.EquitySnapshot
		if body.Hours > 0 {
			start := now.Add(-time.Duration(body.Hours) * time.Hour)
			snaps, err = st.Equity().GetByTimeRange(id, start, now)
		} else {
			snaps, err = st.Equity().GetLatest(id, 2000)
		}
		if err != nil {
			histories[id] = []gin.H{}
			continue
		}
		histories[id] = equityPointsFromSnapshots(snaps, s.traderInitialBalance(st, id))
	}
	writeJSON(c, http.StatusOK, gin.H{"histories": histories})
}

func (s *Service) consolePositionsHistory(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{
			"positions":       []any{},
			"stats":           nil,
			"symbol_stats":    []any{},
			"direction_stats": []any{},
		})
		return
	}
	id := s.resolveOwnedTraderID(c)
	if id == "" {
		writeJSON(c, http.StatusOK, gin.H{
			"positions":       []any{},
			"stats":           nil,
			"symbol_stats":    []any{},
			"direction_stats": []any{},
		})
		return
	}
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	positions, err := st.Position().GetClosedPositions(id, limit)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	if positions == nil {
		positions = []*store.TraderPosition{}
	}

	initial := s.traderInitialBalance(st, id)
	stats, _ := st.Position().GetFullStats(id, initial)
	symbolStats, _ := st.Position().GetSymbolStats(id, 20)
	directionStats, _ := st.Position().GetDirectionStats(id)
	if symbolStats == nil {
		symbolStats = []store.SymbolStats{}
	}
	if directionStats == nil {
		directionStats = []store.DirectionStats{}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"positions":       positions,
		"stats":           stats,
		"symbol_stats":    symbolStats,
		"direction_stats": directionStats,
	})
}

// enrichDecisionFromEquity fills account_state from the nearest equity snapshot
// so DecisionCard can show balance context without a schema migration.
func enrichDecisionFromEquity(st *store.Store, rec *store.DecisionRecord) {
	if st == nil || rec == nil || rec.TraderID == "" || rec.Timestamp.IsZero() {
		return
	}
	if rec.AccountState.TotalBalance > 0 || rec.AccountState.AvailableBalance > 0 {
		return
	}
	start := rec.Timestamp.Add(-3 * time.Minute)
	end := rec.Timestamp.Add(3 * time.Minute)
	snaps, err := st.Equity().GetByTimeRange(rec.TraderID, start, end)
	if err != nil || len(snaps) == 0 {
		// Fallback: latest snapshot before the decision.
		latest, lerr := st.Equity().GetLatest(rec.TraderID, 1)
		if lerr != nil || len(latest) == 0 {
			return
		}
		snaps = latest
	}
	snap := snaps[len(snaps)-1]
	initial := 0.0
	if tr, err := st.Trader().GetByID(rec.TraderID); err == nil && tr != nil {
		initial = tr.InitialBalance
	}
	rec.AccountState = store.AccountSnapshot{
		TotalBalance:          snap.TotalEquity,
		AvailableBalance:      snap.AvailableBalance,
		TotalUnrealizedProfit: snap.UnrealizedPnL,
		PositionCount:         snap.PositionCount,
		MarginUsedPct:         snap.MarginUsedPct,
		InitialBalance:        initial,
	}
}
