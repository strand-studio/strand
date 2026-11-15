package live

import (
	"testing"
	"time"

	"strand/internal/planning/core"
)

func TestBuildIntentOrdersLimitUsesEmptyType(t *testing.T) {
	now := time.Now().UTC()
	plan := testPlan(now)
	orders, err := buildIntentOrders(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) < 3 {
		t.Fatalf("len=%d", len(orders))
	}
	if orders[0].Role != "entry" || orders[0].Type != "" {
		t.Fatalf("entry type=%q want empty for limit compatibility", orders[0].Type)
	}
	for _, o := range orders[1:] {
		if o.Role == "entry" {
			continue
		}
		if o.Type != "" {
			t.Fatalf("protection type=%q want empty", o.Type)
		}
		if o.Status != "pending_parent" {
			t.Fatalf("protection status=%s", o.Status)
		}
	}
}

func TestBuildIntentOrdersMarketType(t *testing.T) {
	now := time.Now().UTC()
	tick := core.Decimal("0.1")
	step := core.Decimal("0.001")
	snap := core.Snapshot{
		Venue: "binance", Symbol: "BTCUSDT", Kind: "perpetual",
		Bid: "100.0", Ask: "100.1", BidSize: "50", AskSize: "50",
		Tick: tick, Step: step, MinQty: "0.001", MinNotional: "5",
		Observed: now, SourceTime: now,
	}
	idea := core.Idea{
		Name: "mkt", Side: "long", Entry: "100.1", Stop: "90.0", Target: "120.0",
		Execution: &core.Execution{Kind: "market"},
	}
	c := core.Constraints{
		Equity: "1000", MaxRisk: "50", MaxNotional: "200",
		MinRewardRisk: "0", FeeBPS: "10", SlippageBPS: "10", Leverage: 1,
	}
	plan, err := core.Compile(snap, c, idea, now)
	if err != nil {
		t.Fatal(err)
	}
	orders, err := buildIntentOrders(plan)
	if err != nil {
		t.Fatal(err)
	}
	if orders[0].Type != "market" {
		t.Fatalf("type=%q want market", orders[0].Type)
	}
}

func TestBuildIntentOrdersLadderAndTWAP(t *testing.T) {
	now := time.Now().UTC()
	tick := core.Decimal("0.1")
	step := core.Decimal("0.001")
	snap := core.Snapshot{
		Venue: "binance", Symbol: "BTCUSDT", Kind: "perpetual",
		Bid: "100.0", Ask: "100.1", BidSize: "50", AskSize: "50",
		Tick: tick, Step: step, MinQty: "0.001", MinNotional: "5",
		Observed: now, SourceTime: now,
	}
	c := core.Constraints{
		Equity: "1000", MaxRisk: "80", MaxNotional: "400",
		MinRewardRisk: "0", FeeBPS: "10", SlippageBPS: "10", Leverage: 1,
	}
	ladderIdea := core.Idea{
		Name: "ladder", Side: "long", Entry: "100.0", Stop: "90.0", Target: "120.0",
		Execution: &core.Execution{
			Kind: "ladder",
			Levels: []core.Level{
				{Price: "100.0", Weight: "0.5"},
				{Price: "99.5", Weight: "0.5"},
			},
		},
	}
	ladder, err := core.Compile(snap, c, ladderIdea, now)
	if err != nil {
		t.Fatal(err)
	}
	orders, err := buildIntentOrders(ladder)
	if err != nil {
		t.Fatal(err)
	}
	entries := 0
	for _, o := range orders {
		if o.Role == "entry" {
			entries++
			if o.Type != "" {
				t.Fatalf("ladder entry type=%q", o.Type)
			}
		}
	}
	if entries < 2 {
		t.Fatalf("ladder entries=%d want >=2", entries)
	}

	twapIdea := core.Idea{
		Name: "twap", Side: "long", Entry: "100.0", Stop: "90.0", Target: "120.0",
		Execution: &core.Execution{Kind: "twap", Slices: 3, IntervalSeconds: 30},
	}
	twap, err := core.Compile(snap, c, twapIdea, now)
	if err != nil {
		t.Fatal(err)
	}
	orders, err = buildIntentOrders(twap)
	if err != nil {
		t.Fatal(err)
	}
	scheduled := 0
	for _, o := range orders {
		if o.Role == "entry" && o.Status == "scheduled" {
			scheduled++
		}
	}
	if scheduled < 1 {
		t.Fatalf("twap scheduled entries=%d", scheduled)
	}
}
