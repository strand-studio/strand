package api

import (
	"testing"
	"time"

	"strand/internal/planning/core"
)

func TestPlanningChatKinds(t *testing.T) {
	if got := planningChatKinds("auto", false); len(got) != 4 {
		t.Fatalf("auto kinds=%v", got)
	}
	if got := planningChatKinds("limit", false); len(got) != 1 || got[0] != "limit" {
		t.Fatalf("limit kinds=%v", got)
	}
	if got := planningChatKinds("market", true); len(got) != 4 {
		t.Fatalf("compare kinds=%v", got)
	}
}

func TestInferPlanningSide(t *testing.T) {
	if inferPlanningSide("open a short", "") != "short" {
		t.Fatal("expected short")
	}
	if inferPlanningSide("做空 eth", "") != "short" {
		t.Fatal("expected short from chinese")
	}
	if inferPlanningSide("buy the dip", core.EntryAI500TrendLong) != "long" {
		t.Fatal("AI500 forces long")
	}
}

func TestBuildPlanningChatIdeaCompiles(t *testing.T) {
	now := time.Now().UTC()
	snap := core.Snapshot{
		Venue: "binance", Symbol: "BTCUSDT", Kind: "perpetual",
		Bid: "100.0", Ask: "100.1", BidSize: "50", AskSize: "50",
		Tick: "0.1", Step: "0.001", MinQty: "0.001", MinNotional: "5",
		Observed: now, SourceTime: now,
	}
	c := core.Constraints{
		Equity: "1000", MaxRisk: "50", MaxNotional: "200",
		MinRewardRisk: "0", FeeBPS: "10", SlippageBPS: "10", Leverage: 1,
	}
	hint := planningAIHint{Side: "long", Rationale: "test", StopFactor: 0.97, TargetFactor: 1.05}
	for _, kind := range []string{"market", "limit", "ladder", "twap"} {
		idea, err := buildPlanningChatIdea(snap, kind, hint)
		if err != nil {
			t.Fatalf("%s idea: %v", kind, err)
		}
		if _, err := core.Compile(snap, c, idea, now); err != nil {
			t.Fatalf("%s compile: %v", kind, err)
		}
	}
}
