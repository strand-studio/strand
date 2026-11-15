package live

import (
	"errors"
	"fmt"
	"time"

	"strand/internal/planning/core"
	"strand/internal/trader/execution"
)

func buildIntentOrders(plan core.Plan) ([]Order, error) {
	specs := plan.Orders
	if len(specs) == 0 {
		specs = []core.OrderSpec{{
			Index:         1,
			Price:         plan.Idea.Entry,
			Quantity:      plan.Quantity,
			VenueQuantity: plan.VenueQuantity,
		}}
	}
	// Empty Type = venue limit entry (Binance/HL legacy). Only market entries
	// set Type explicitly; "limit" string is rejected by those SubmitOrder gates.
	kind := ""
	if plan.Idea.Execution != nil && plan.Idea.Execution.Kind == "market" {
		kind = "market"
	}
	orders := make([]Order, 0, len(specs)+2)
	var entryParent string
	for i, spec := range specs {
		clientID, err := execution.NewClientID()
		if err != nil {
			return nil, err
		}
		if i == 0 {
			entryParent = clientID
		}
		qty := spec.Quantity
		if spec.VenueQuantity != "" {
			qty = spec.VenueQuantity
		}
		status := "NEW"
		if spec.OffsetSeconds > 0 {
			status = "scheduled"
		}
		orders = append(orders, Order{
			Type:     kind,
			ClientID: clientID,
			Role:     "entry",
			Symbol:   plan.Snapshot.Symbol,
			Side:     plan.Idea.Side,
			Price:    execution.Decimal(spec.Price),
			Quantity: execution.Decimal(qty),
			Status:   status,
			Filled:   "0",
		})
	}
	if !execution.Positive(execution.Decimal(plan.Idea.Stop)) || !execution.Positive(execution.Decimal(plan.Idea.Target)) {
		return nil, errors.New("approved plan is missing stop or target")
	}
	totalQty := plan.Quantity
	if plan.VenueQuantity != "" {
		totalQty = plan.VenueQuantity
	}
	for _, role := range []string{"sl", "tp"} {
		clientID, err := execution.NewClientID()
		if err != nil {
			return nil, err
		}
		price := plan.Idea.Stop
		if role == "tp" {
			price = plan.Idea.Target
		}
		orders = append(orders, Order{
			ClientID: clientID,
			ParentID: entryParent,
			Role:     role,
			Symbol:   plan.Snapshot.Symbol,
			Side:     plan.Idea.Side,
			Price:    execution.Decimal(price),
			Quantity: execution.Decimal(totalQty),
			Status:   "pending_parent",
			Filled:   "0",
		})
	}
	return orders, nil
}

func routePersisted(b Broker, orders []Order) ([]Order, error) {
	out := make([]Order, len(orders))
	for i, o := range orders {
		routed, err := routedOrder(b, o)
		if err != nil {
			return nil, fmt.Errorf("route client id: %w", err)
		}
		out[i] = routed
	}
	return out, nil
}

func hasEntryFill(orders []Order) bool {
	for _, o := range orders {
		if o.Role == "entry" && o.Filled != "" && o.Filled != "0" {
			return true
		}
	}
	return false
}

func anyEntryWorking(orders []Order) bool {
	for _, o := range orders {
		if o.Role != "entry" {
			continue
		}
		switch o.Status {
		case "NEW", "PARTIALLY_FILLED", "submitting", "UNKNOWN", "TRIGGERED", "scheduled":
			return true
		}
	}
	return false
}

func allEntriesTerminalEmpty(orders []Order) bool {
	saw := false
	for _, o := range orders {
		if o.Role != "entry" {
			continue
		}
		saw = true
		switch o.Status {
		case "CANCELED", "CANCELLED", "EXPIRED", "REJECTED":
			if o.Filled != "" && o.Filled != "0" {
				return false
			}
		default:
			return false
		}
	}
	return saw
}

func protectionsPending(orders []Order) bool {
	for _, o := range orders {
		if (o.Role == "sl" || o.Role == "tp") && (o.Status == "pending_parent" || o.Status == "") {
			return true
		}
	}
	return false
}

func entryOffset(plan core.Plan, entryOrdinal int) int {
	if len(plan.Orders) == 0 || entryOrdinal < 0 {
		return 0
	}
	if entryOrdinal >= len(plan.Orders) {
		return plan.Orders[len(plan.Orders)-1].OffsetSeconds
	}
	return plan.Orders[entryOrdinal].OffsetSeconds
}

func dueScheduledIndices(orders []Order, startedAt time.Time, now time.Time, plan core.Plan) []int {
	if startedAt.IsZero() {
		return nil
	}
	var due []int
	entryOrdinal := 0
	for i, o := range orders {
		if o.Role != "entry" {
			continue
		}
		offset := entryOffset(plan, entryOrdinal)
		entryOrdinal++
		if o.Status != "scheduled" {
			continue
		}
		if !now.Before(startedAt.Add(time.Duration(offset) * time.Second)) {
			due = append(due, i)
		}
	}
	return due
}
