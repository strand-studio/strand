package live

import (
	"context"

	"strand/internal/trader/execution"
)

// AdaptTrader exposes an exchange execution.Trader as a planning Broker.
func AdaptTrader(t execution.Trader) Broker {
	return &traderBroker{t: t}
}

type traderBroker struct {
	t execution.Trader
}

func (b *traderBroker) Venue() string { return b.t.Venue() }

func (b *traderBroker) Account(ctx context.Context, symbol string) (Account, error) {
	return b.t.AccountSnapshot(ctx, symbol)
}

func (b *traderBroker) Prepare(ctx context.Context, symbol string) error {
	return b.t.PrepareExecution(ctx, symbol)
}

func (b *traderBroker) Place(ctx context.Context, o Order) (Order, error) {
	return b.t.SubmitOrder(ctx, o)
}

func (b *traderBroker) Lookup(ctx context.Context, o Order) (Order, error) {
	return b.t.LookupOrder(ctx, o)
}

func (b *traderBroker) Cancel(ctx context.Context, o Order) error {
	return b.t.CancelExecutionOrder(ctx, o)
}

// PrepareLeverage sets venue leverage when the underlying trader supports it.
func (b *traderBroker) PrepareLeverage(ctx context.Context, symbol string, leverage int) error {
	type leverageSetter interface {
		PrepareExecutionLeverage(context.Context, string, int) error
	}
	if setter, ok := b.t.(leverageSetter); ok {
		return setter.PrepareExecutionLeverage(ctx, symbol, leverage)
	}
	return b.t.PrepareExecution(ctx, symbol)
}

type leverageBroker interface {
	PrepareLeverage(context.Context, string, int) error
}
