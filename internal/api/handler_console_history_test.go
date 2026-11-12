package api

import (
	"testing"
	"time"

	"strand/internal/store"
)

func TestEquityPointsFromSnapshots(t *testing.T) {
	base := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	snaps := []*store.EquitySnapshot{
		{Timestamp: base, TotalEquity: 1000, AvailableBalance: 800, UnrealizedPnL: 10, PositionCount: 1},
		{Timestamp: base.Add(time.Hour), TotalEquity: 1100, AvailableBalance: 850, UnrealizedPnL: 20, PositionCount: 2},
	}
	points := equityPointsFromSnapshots(snaps, 1000)
	if len(points) != 2 {
		t.Fatalf("len=%d", len(points))
	}
	if points[1]["pnl"].(float64) != 100 {
		t.Fatalf("pnl=%v", points[1]["pnl"])
	}
	if points[1]["pnl_pct"].(float64) != 10 {
		t.Fatalf("pnl_pct=%v", points[1]["pnl_pct"])
	}
	if points[0]["cycle_number"].(int) != 1 {
		t.Fatalf("cycle=%v", points[0]["cycle_number"])
	}
}

func TestEnrichDecisionFromEquityNilSafe(t *testing.T) {
	enrichDecisionFromEquity(nil, nil)
	enrichDecisionFromEquity(nil, &store.DecisionRecord{})
}
