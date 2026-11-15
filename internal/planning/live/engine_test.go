package live

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"strand/internal/planning/core"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockBroker struct {
	venue   string
	placed  map[string]Order
	lookups map[string]Order
	failOn  string
}

func (m *mockBroker) Venue() string { return m.venue }
func (m *mockBroker) Account(context.Context, string) (Account, error) {
	return Account{Equity: "1000", Available: "1000", FeeBPS: "5", Position: "0"}, nil
}
func (m *mockBroker) Prepare(context.Context, string) error { return nil }
func (m *mockBroker) PrepareLeverage(context.Context, string, int) error {
	return nil
}
func (m *mockBroker) Place(_ context.Context, o Order) (Order, error) {
	if m.failOn == o.Role {
		return o, errors.New("submission unconfirmed")
	}
	o.OrderID = "ex-" + o.ClientID[:8]
	if o.Type == "market" && o.Role == "entry" {
		o.Status = "FILLED"
		o.Filled = o.Quantity
		o.Average = o.Price
	} else {
		o.Status = "NEW"
		o.Filled = "0"
	}
	if m.placed == nil {
		m.placed = map[string]Order{}
	}
	m.placed[o.ClientID] = o
	return o, nil
}
func (m *mockBroker) Lookup(_ context.Context, o Order) (Order, error) {
	if got, ok := m.lookups[o.ClientID]; ok {
		return got, nil
	}
	if got, ok := m.placed[o.ClientID]; ok {
		return got, nil
	}
	return o, errors.New("lookup unconfirmed")
}
func (m *mockBroker) Cancel(_ context.Context, o Order) error {
	if m.placed != nil {
		got := m.placed[o.ClientID]
		got.Status = "CANCELED"
		m.placed[o.ClientID] = got
	}
	return nil
}
func (m *mockBroker) RouteClientID(id string) (string, error) {
	return "route-" + id, nil
}

func testPlan(now time.Time) core.Plan {
	tick := core.Decimal("0.1")
	step := core.Decimal("0.001")
	snap := core.Snapshot{
		Venue: "binance", Symbol: "BTCUSDT", Kind: "perpetual",
		Bid: "100.0", Ask: "100.1", BidSize: "10", AskSize: "10",
		Tick: tick, Step: step, MinQty: "0.001", MinNotional: "5",
		Observed: now, SourceTime: now,
	}
	idea := core.Idea{
		Name: "test", Side: "long",
		Entry: "100.0", Stop: "90.0", Target: "120.0",
		Rationale: "unit",
		Execution: &core.Execution{Kind: "limit"},
	}
	c := core.Constraints{
		Equity: "1000", MaxRisk: "50", MaxNotional: "200",
		MinRewardRisk: "0", FeeBPS: "10", SlippageBPS: "10",
		Leverage: 1,
	}
	p, err := core.Compile(snap, c, idea, now)
	if err != nil {
		panic(err)
	}
	return p
}

func testEngine(t *testing.T, broker *mockBroker) *Engine {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "live.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Run{}, &Lease{}); err != nil {
		t.Fatal(err)
	}
	factory := func(context.Context, string, string) (Broker, string, error) {
		return broker, "acct-key", nil
	}
	return NewEngine(db, factory, nil)
}

func TestApproveExecuteStop(t *testing.T) {
	now := time.Now().UTC()
	plan := testPlan(now)
	broker := &mockBroker{venue: "binance"}
	eng := testEngine(t, broker)

	view, err := eng.Approve(context.Background(), ApproveInput{
		UserID: "u1", ExchangeID: "ex1", Plan: plan, Hash: plan.Hash, Confirmed: true, Now: now,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if view.State != "approved" || view.ApprovalHash == "" || len(view.Orders) < 3 {
		t.Fatalf("unexpected approve view: %+v", view)
	}
	if view.Orders[0].VenueClientID == "" {
		t.Fatal("expected routed venue client id before submit")
	}

	_, err = eng.Approve(context.Background(), ApproveInput{
		UserID: "u1", ExchangeID: "ex1", Plan: plan, Hash: plan.Hash, Confirmed: true, Now: now,
	})
	if !errors.Is(err, ErrUnresolvedPlan) {
		t.Fatalf("expected unresolved plan, got %v", err)
	}

	executed, err := eng.Execute(context.Background(), ExecuteInput{
		UserID: "u1", RunID: view.ID, ApprovalHash: view.ApprovalHash, Now: now,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if executed.State != "running" && executed.State != "position_open" {
		t.Fatalf("state=%s want running/position_open", executed.State)
	}
	if broker.placed[executed.Orders[0].ClientID].OrderID == "" {
		t.Fatal("entry was not placed")
	}

	stopped, err := eng.Stop(context.Background(), "u1", executed.ID, executed.ApprovalHash)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped.State != "cancelled" && stopped.State != "needs_attention" && stopped.State != "position_open" {
		// limit may cancel cleanly
		if !isTerminal(stopped.State) && stopped.State != "stopping" {
			t.Fatalf("unexpected stop state %s", stopped.State)
		}
	}
}

func TestReplaceApproval(t *testing.T) {
	now := time.Now().UTC()
	plan := testPlan(now)
	broker := &mockBroker{venue: "binance"}
	eng := testEngine(t, broker)

	first, err := eng.Approve(context.Background(), ApproveInput{
		UserID: "u1", ExchangeID: "ex1", Plan: plan, Hash: plan.Hash, Confirmed: true, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := eng.Approve(context.Background(), ApproveInput{
		UserID: "u1", ExchangeID: "ex1", Plan: plan, Hash: plan.Hash, Confirmed: true, Now: now,
		ReplaceApprovalID: first.ID, ReplaceApprovalHash: first.ApprovalHash,
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("expected new run id")
	}
	prev, err := eng.loadRun("u1", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if prev.State != "cancelled" {
		t.Fatalf("replaced run state=%s", prev.State)
	}
}

func TestMarketExecutePlacesProtections(t *testing.T) {
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
	broker := &mockBroker{venue: "binance"}
	eng := testEngine(t, broker)
	view, err := eng.Approve(context.Background(), ApproveInput{
		UserID: "u1", ExchangeID: "ex1", Plan: plan, Hash: plan.Hash, Confirmed: true, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	executed, err := eng.Execute(context.Background(), ExecuteInput{
		UserID: "u1", RunID: view.ID, ApprovalHash: view.ApprovalHash, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	var roles []string
	for _, o := range executed.Orders {
		if o.OrderID != "" {
			roles = append(roles, o.Role)
		}
	}
	if len(roles) < 3 {
		t.Fatalf("expected entry+sl+tp placements, got %v state=%s", roles, executed.State)
	}
	if executed.State != "position_open" && executed.State != "running" {
		t.Fatalf("state=%s", executed.State)
	}
}
