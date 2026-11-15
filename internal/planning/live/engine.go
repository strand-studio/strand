package live

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"strand/internal/planning"
	"strand/internal/planning/core"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrUnresolvedPlan = errors.New("already has an approved or unresolved plan")
	ErrAccountLease   = errors.New("An earlier trade is reserving this account")
	ErrNotFound       = errors.New("planning run not found")
	ErrHashMismatch   = errors.New("approval hash mismatch")
	ErrInvalidState   = errors.New("run is not in a valid state for this action")
)

// SnapshotFunc loads a fresh book for pre-submit slippage checks.
type SnapshotFunc func(ctx context.Context, userID, exchangeID, venue, symbol string) (core.Snapshot, error)

type Engine struct {
	DB       *gorm.DB
	Factory  Factory
	Snapshot SnapshotFunc
}

func NewEngine(db *gorm.DB, factory Factory, snap SnapshotFunc) *Engine {
	return &Engine{DB: db, Factory: factory, Snapshot: snap}
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func approvalDigest(runID, planHash string, at time.Time) string {
	sum := sha256.Sum256([]byte(runID + "|" + planHash + "|" + at.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:])
}

func encodePlan(p core.Plan) (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodePlan(raw string) (core.Plan, error) {
	var p core.Plan
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return p, err
	}
	return p, nil
}

func encodeOrders(orders []Order) (string, error) {
	b, err := json.Marshal(orders)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeOrders(raw string) ([]Order, error) {
	if strings.TrimSpace(raw) == "" {
		return []Order{}, nil
	}
	var orders []Order
	if err := json.Unmarshal([]byte(raw), &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (e *Engine) viewOf(run Run) (View, error) {
	plan, err := decodePlan(run.PlanJSON)
	if err != nil {
		return View{}, err
	}
	orders, err := decodeOrders(run.OrdersJSON)
	if err != nil {
		return View{}, err
	}
	return View{Run: run, Plan: plan, Orders: orders}, nil
}

func (e *Engine) saveView(v View) error {
	planJSON, err := encodePlan(v.Plan)
	if err != nil {
		return err
	}
	ordersJSON, err := encodeOrders(v.Orders)
	if err != nil {
		return err
	}
	v.PlanJSON = planJSON
	v.OrdersJSON = ordersJSON
	v.UpdatedAt = time.Now().UTC()
	return e.DB.Save(&v.Run).Error
}

func (e *Engine) loadRun(userID, runID string) (View, error) {
	var run Run
	if err := e.DB.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return View{}, ErrNotFound
		}
		return View{}, err
	}
	return e.viewOf(run)
}

func activeStates() []string {
	return []string{"approved", "starting", "running", "position_open", "stopping", "needs_attention"}
}

func isTerminal(state string) bool {
	switch state {
	case "completed", "cancelled", "expired":
		return true
	default:
		return false
	}
}

func (e *Engine) ListViews(userID string) ([]View, error) {
	var runs []Run
	if err := e.DB.Where("user_id = ?", userID).Order("updated_at desc").Limit(100).Find(&runs).Error; err != nil {
		return nil, err
	}
	out := make([]View, 0, len(runs))
	for _, run := range runs {
		v, err := e.viewOf(run)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (e *Engine) AccountsMap(userID string) (map[string]string, error) {
	views, err := e.ListViews(userID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, v := range views {
		if v.PlanID != "" && v.ExchangeID != "" {
			out[v.PlanID] = v.ExchangeID
		}
	}
	return out, nil
}

type ApproveInput struct {
	UserID              string
	ExchangeID          string
	Plan                core.Plan
	Hash                string
	Confirmed           bool
	ReplaceApprovalID   string
	ReplaceApprovalHash string
	Now                 time.Time
}

func (e *Engine) Approve(ctx context.Context, in ApproveInput) (View, error) {
	if e.DB == nil || e.Factory == nil {
		return View{}, errors.New("planning live engine is not configured")
	}
	if !in.Confirmed {
		return View{}, errors.New("confirmation required to approve a live plan")
	}
	if strings.TrimSpace(in.ExchangeID) == "" {
		return View{}, errors.New("exchange account required before approving a live plan")
	}
	if in.Plan.Constraints.PlanningOnly {
		return View{}, errors.New("this plan is an equity preview only; fund available margin before approving")
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if in.Hash != "" && in.Hash != in.Plan.Hash {
		return View{}, errors.New("plan hash mismatch")
	}
	if err := in.Plan.Verify(now); err != nil {
		return View{}, err
	}

	broker, accountKey, err := e.Factory(ctx, in.UserID, in.ExchangeID)
	if err != nil {
		return View{}, err
	}
	if broker.Venue() != "" && broker.Venue() != in.Plan.Snapshot.Venue {
		return View{}, fmt.Errorf("exchange venue %s does not match plan venue %s", broker.Venue(), in.Plan.Snapshot.Venue)
	}

	unlock := planning.LockAccount(accountKey)
	defer unlock()

	if in.ReplaceApprovalID != "" {
		prev, err := e.loadRun(in.UserID, in.ReplaceApprovalID)
		if err != nil {
			return View{}, err
		}
		if prev.ApprovalHash != in.ReplaceApprovalHash {
			return View{}, ErrHashMismatch
		}
		if prev.State != "approved" {
			return View{}, errors.New("only an unused approved plan can be replaced")
		}
		if prev.ExchangeID != in.ExchangeID {
			return View{}, errors.New("replacement approval must use the same exchange account")
		}
		prev.State = "cancelled"
		prev.Error = "replaced by a newer approval"
		if err := e.saveView(prev); err != nil {
			return View{}, err
		}
		_ = e.DB.Where("run_id = ?", prev.ID).Delete(&Lease{}).Error
	}

	var open int64
	if err := e.DB.Model(&Run{}).
		Where("user_id = ? AND exchange_id = ? AND state IN ?", in.UserID, in.ExchangeID, activeStates()).
		Count(&open).Error; err != nil {
		return View{}, err
	}
	if open > 0 {
		return View{}, ErrUnresolvedPlan
	}

	var leaseCount int64
	if err := e.DB.Model(&Lease{}).Where("account_key = ?", accountKey).Count(&leaseCount).Error; err != nil {
		return View{}, err
	}
	if leaseCount > 0 {
		return View{}, ErrAccountLease
	}

	runID, err := newID()
	if err != nil {
		return View{}, err
	}
	approvedAt := now
	orders, err := buildIntentOrders(in.Plan)
	if err != nil {
		return View{}, err
	}
	orders, err = routePersisted(broker, orders)
	if err != nil {
		return View{}, err
	}
	deadline := in.Plan.Expires
	view := View{
		Run: Run{
			ID:           runID,
			UserID:       in.UserID,
			ExchangeID:   in.ExchangeID,
			AccountKey:   accountKey,
			PlanID:       in.Plan.ID,
			PlanHash:     in.Plan.Hash,
			ApprovalHash: approvalDigest(runID, in.Plan.Hash, approvedAt),
			State:        "approved",
			ApprovedAt:   approvedAt,
			Deadline:     &deadline,
			UpdatedAt:    approvedAt,
		},
		Plan:   in.Plan,
		Orders: orders,
	}
	if err := e.saveView(view); err != nil {
		return View{}, err
	}
	lease := Lease{
		AccountKey: accountKey,
		RunID:      runID,
		UserID:     in.UserID,
		ExchangeID: in.ExchangeID,
	}
	if err := e.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&lease).Error; err != nil {
		return View{}, err
	}
	var confirm int64
	if err := e.DB.Model(&Lease{}).Where("account_key = ? AND run_id = ?", accountKey, runID).Count(&confirm).Error; err != nil || confirm == 0 {
		view.State = "cancelled"
		view.Error = "failed to reserve account lease"
		_ = e.saveView(view)
		return View{}, ErrAccountLease
	}
	return view, nil
}

type ExecuteInput struct {
	UserID       string
	RunID        string
	ApprovalHash string
	Now          time.Time
}

func (e *Engine) Execute(ctx context.Context, in ExecuteInput) (View, error) {
	view, err := e.loadRun(in.UserID, in.RunID)
	if err != nil {
		return View{}, err
	}
	if view.ApprovalHash != in.ApprovalHash {
		return View{}, ErrHashMismatch
	}
	if view.State != "approved" {
		return View{}, ErrInvalidState
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !now.Before(view.Plan.Expires) {
		view.State = "expired"
		view.Error = "plan expired; replan and approve again"
		_ = e.releaseLease(view.ID)
		_ = e.saveView(view)
		return view, errors.New("plan expired; replan and approve again")
	}

	unlock := planning.LockAccount(view.AccountKey)
	defer unlock()

	broker, _, err := e.Factory(ctx, in.UserID, view.ExchangeID)
	if err != nil {
		return View{}, err
	}

	if e.Snapshot != nil {
		snap, serr := e.Snapshot(ctx, in.UserID, view.ExchangeID, view.Plan.Snapshot.Venue, view.Plan.Snapshot.Symbol)
		if serr != nil {
			return View{}, serr
		}
		if cerr := core.CheckBook(view.Plan, snap, now); cerr != nil {
			view.State = "needs_attention"
			view.Error = cerr.Error()
			_ = e.saveView(view)
			return view, cerr
		}
	}

	view.State = "starting"
	started := now
	view.StartedAt = &started
	if err := e.saveView(view); err != nil {
		return View{}, err
	}

	lev := view.Plan.Constraints.EffectiveLeverage()
	if lb, ok := broker.(leverageBroker); ok {
		if err := lb.PrepareLeverage(ctx, view.Plan.Snapshot.Symbol, lev); err != nil {
			view.State = "needs_attention"
			view.Error = err.Error()
			_ = e.saveView(view)
			return view, err
		}
	} else if err := broker.Prepare(ctx, view.Plan.Snapshot.Symbol); err != nil {
		view.State = "needs_attention"
		view.Error = err.Error()
		_ = e.saveView(view)
		return view, err
	}

	if err := e.submitDueEntries(ctx, broker, &view, now); err != nil {
		view.State = "needs_attention"
		if view.Error == "" {
			view.Error = err.Error()
		}
		_ = e.saveView(view)
		return view, err
	}
	if err := e.maybePlaceProtections(ctx, broker, &view); err != nil {
		view.State = "needs_attention"
		view.Error = err.Error()
		_ = e.saveView(view)
		return view, err
	}
	e.deriveState(&view, now)
	if err := e.saveView(view); err != nil {
		return View{}, err
	}
	if isTerminal(view.State) {
		_ = e.releaseLease(view.ID)
	}
	return view, nil
}

func (e *Engine) CheckStatus(ctx context.Context, userID, runID string) (View, error) {
	return e.refresh(ctx, userID, runID, "", false)
}

func (e *Engine) Reconcile(ctx context.Context, userID, runID, approvalHash string) (View, error) {
	return e.refresh(ctx, userID, runID, approvalHash, false)
}

func (e *Engine) Stop(ctx context.Context, userID, runID, approvalHash string) (View, error) {
	view, err := e.loadRun(userID, runID)
	if err != nil {
		return View{}, err
	}
	if approvalHash != "" && view.ApprovalHash != approvalHash {
		return View{}, ErrHashMismatch
	}
	if isTerminal(view.State) {
		return view, nil
	}

	unlock := planning.LockAccount(view.AccountKey)
	defer unlock()

	now := time.Now().UTC()
	if view.State == "approved" {
		view.State = "cancelled"
		view.Error = "approval revoked before submission"
		for i := range view.Orders {
			if view.Orders[i].Status == "scheduled" || view.Orders[i].Status == "NEW" || view.Orders[i].Status == "pending_parent" {
				view.Orders[i].Status = "CANCELED"
			}
		}
		_ = e.releaseLease(view.ID)
		if err := e.saveView(view); err != nil {
			return View{}, err
		}
		return view, nil
	}

	view.State = "stopping"
	_ = e.saveView(view)

	broker, _, err := e.Factory(ctx, userID, view.ExchangeID)
	if err != nil {
		view.State = "needs_attention"
		view.Error = err.Error()
		_ = e.saveView(view)
		return view, err
	}

	for i, o := range view.Orders {
		if o.Role != "entry" {
			continue
		}
		switch o.Status {
		case "scheduled", "pending_parent":
			view.Orders[i].Status = "CANCELED"
			continue
		case "NEW", "PARTIALLY_FILLED", "submitting", "UNKNOWN":
			if err := broker.Cancel(ctx, o); err != nil {
				view.Orders[i].Status = "UNKNOWN"
				view.Orders[i].Error = err.Error()
				view.Error = "entry cancellation is unconfirmed; reconcile before retrying"
				view.State = "needs_attention"
				_ = e.saveView(view)
				return view, errors.New(view.Error)
			}
			looked, lerr := broker.Lookup(ctx, o)
			if lerr != nil {
				view.Orders[i].Status = "UNKNOWN"
				view.Orders[i].Error = lerr.Error()
			} else {
				view.Orders[i] = looked
			}
		}
	}

	// Refresh lookups then derive.
	view, err = e.refreshLocked(ctx, broker, view, now, true)
	if err != nil {
		return view, err
	}
	if !isTerminal(view.State) && !hasEntryFill(view.Orders) {
		view.State = "cancelled"
		_ = e.releaseLease(view.ID)
		_ = e.saveView(view)
	}
	return view, nil
}

func (e *Engine) refresh(ctx context.Context, userID, runID, approvalHash string, stopping bool) (View, error) {
	view, err := e.loadRun(userID, runID)
	if err != nil {
		return View{}, err
	}
	if approvalHash != "" && view.ApprovalHash != approvalHash {
		return View{}, ErrHashMismatch
	}
	if isTerminal(view.State) {
		return view, nil
	}
	unlock := planning.LockAccount(view.AccountKey)
	defer unlock()

	broker, _, err := e.Factory(ctx, userID, view.ExchangeID)
	if err != nil {
		return View{}, err
	}
	return e.refreshLocked(ctx, broker, view, time.Now().UTC(), stopping)
}

func (e *Engine) refreshLocked(ctx context.Context, broker Broker, view View, now time.Time, stopping bool) (View, error) {
	if view.State == "approved" {
		if !now.Before(view.Plan.Expires) {
			view.State = "expired"
			view.Error = "plan expired; replan and approve again"
			_ = e.releaseLease(view.ID)
			_ = e.saveView(view)
		}
		return view, nil
	}

	// Submit due TWAP slices (never during pure stop cancel path before cancel).
	if !stopping {
		if err := e.submitDueEntries(ctx, broker, &view, now); err != nil {
			view.State = "needs_attention"
			view.Error = err.Error()
			_ = e.saveView(view)
			return view, err
		}
	}

	for i, o := range view.Orders {
		if o.ClientID == "" {
			continue
		}
		switch o.Status {
		case "scheduled", "pending_parent", "CANCELED", "CANCELLED", "REJECTED", "EXPIRED":
			continue
		}
		// Never look up intents that have not been submitted. Missing ≠ filled.
		if o.OrderID == "" && o.Status != "UNKNOWN" && o.Status != "submitting" {
			continue
		}
		looked, err := broker.Lookup(ctx, o)
		if err != nil {
			view.Orders[i].Status = "UNKNOWN"
			view.Orders[i].Error = err.Error()
			view.State = "needs_attention"
			view.Error = err.Error()
			_ = e.saveView(view)
			return view, err
		}
		view.Orders[i] = looked
	}

	if !stopping {
		if err := e.maybePlaceProtections(ctx, broker, &view); err != nil {
			view.State = "needs_attention"
			view.Error = err.Error()
			_ = e.saveView(view)
			return view, err
		}
	}

	if stopping {
		view.State = "stopping"
	}
	e.deriveState(&view, now)
	if err := e.saveView(view); err != nil {
		return View{}, err
	}
	if isTerminal(view.State) {
		_ = e.releaseLease(view.ID)
	}
	return view, nil
}

func (e *Engine) submitDueEntries(ctx context.Context, broker Broker, view *View, now time.Time) error {
	started := now
	if view.StartedAt != nil {
		started = *view.StartedAt
	}
	// Immediate entries (offset 0) plus due scheduled.
	for i, o := range view.Orders {
		if o.Role != "entry" {
			continue
		}
		if o.Status == "NEW" && o.OrderID == "" {
			// treat as ready to submit (offset 0)
		} else if o.Status == "scheduled" {
			continue
		} else {
			continue
		}
		if err := e.placeEntry(ctx, broker, view, i); err != nil {
			return err
		}
	}
	for _, idx := range dueScheduledIndices(view.Orders, started, now, view.Plan) {
		if err := e.placeEntry(ctx, broker, view, idx); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) placeEntry(ctx context.Context, broker Broker, view *View, idx int) error {
	o := view.Orders[idx]
	o.Status = "submitting"
	view.Orders[idx] = o
	// Persist intent status before the network call.
	if err := e.saveView(*view); err != nil {
		return err
	}
	placed, err := broker.Place(ctx, o)
	if err != nil {
		view.Orders[idx].Status = "UNKNOWN"
		view.Orders[idx].Error = err.Error()
		view.Error = err.Error()
		_ = e.saveView(*view)
		return err
	}
	view.Orders[idx] = placed
	return e.saveView(*view)
}

func (e *Engine) maybePlaceProtections(ctx context.Context, broker Broker, view *View) error {
	if !hasEntryFill(view.Orders) || !protectionsPending(view.Orders) {
		return nil
	}
	for i, o := range view.Orders {
		if o.Role != "sl" && o.Role != "tp" {
			continue
		}
		if o.Status != "pending_parent" && o.Status != "" {
			continue
		}
		o.Status = "submitting"
		view.Orders[i] = o
		if err := e.saveView(*view); err != nil {
			return err
		}
		placed, err := broker.Place(ctx, o)
		if err != nil {
			view.Orders[i].Status = "UNKNOWN"
			view.Orders[i].Error = err.Error()
			_ = e.saveView(*view)
			return err
		}
		view.Orders[i] = placed
		if err := e.saveView(*view); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) deriveState(view *View, now time.Time) {
	if view.State == "needs_attention" && view.Error != "" {
		// Keep attention until reconcile clears errors.
		clean := true
		for _, o := range view.Orders {
			if o.Status == "UNKNOWN" || o.Error != "" {
				clean = false
				break
			}
		}
		if !clean {
			return
		}
	}

	if allEntriesTerminalEmpty(view.Orders) {
		view.State = "cancelled"
		return
	}

	filled := hasEntryFill(view.Orders)
	working := anyEntryWorking(view.Orders)
	acctPos := false
	if filled {
		acctPos = true
	}

	// Protective fills closing the basket.
	slFilled, tpFilled := false, false
	for _, o := range view.Orders {
		if o.Role == "sl" && o.Filled != "" && o.Filled != "0" {
			slFilled = true
		}
		if o.Role == "tp" && o.Filled != "" && o.Filled != "0" {
			tpFilled = true
		}
	}
	if filled && (slFilled || tpFilled) && !working {
		view.State = "completed"
		return
	}
	if filled && !working && !protectionsPending(view.Orders) {
		view.State = "position_open"
		return
	}
	if filled || acctPos {
		if working {
			view.State = "running"
		} else {
			view.State = "position_open"
		}
		return
	}
	if working {
		view.State = "running"
		return
	}
	if view.Deadline != nil && !now.Before(*view.Deadline) && !filled {
		view.State = "expired"
		return
	}
	if view.State == "starting" || view.State == "stopping" {
		return
	}
	view.State = "running"
}

func (e *Engine) releaseLease(runID string) error {
	return e.DB.Where("run_id = ?", runID).Delete(&Lease{}).Error
}
