package live

import (
	"errors"
	"math"
	"strings"
	"time"
)

// MarkSymbolClosed completes active runs for an exchange symbol after a manual close.
func (e *Engine) MarkSymbolClosed(userID, exchangeID, symbol string) error {
	if e.DB == nil {
		return errors.New("planning live engine is not configured")
	}
	symbol = strings.TrimSpace(symbol)
	var runs []Run
	if err := e.DB.Where("user_id = ? AND exchange_id = ? AND state IN ?", userID, exchangeID, activeStates()).
		Find(&runs).Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, run := range runs {
		view, err := e.viewOf(run)
		if err != nil {
			return err
		}
		if !SamePlanningSymbol(view.Plan.Snapshot.Symbol, symbol) {
			continue
		}
		view.State = "completed"
		view.Error = "position closed manually from Trade"
		view.UpdatedAt = now
		for i := range view.Orders {
			switch view.Orders[i].Status {
			case "scheduled", "pending_parent", "NEW", "submitting":
				if view.Orders[i].OrderID == "" {
					view.Orders[i].Status = "CANCELED"
				}
			}
		}
		if err := e.saveView(view); err != nil {
			return err
		}
		_ = e.releaseLease(view.ID)
	}
	return nil
}

func SamePlanningSymbol(a, b string) bool {
	na := normalizePlanningSymbol(a)
	nb := normalizePlanningSymbol(b)
	return na != "" && na == nb
}

func normalizePlanningSymbol(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	s = strings.TrimPrefix(s, "XYZ:")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.TrimSuffix(s, "SWAP")
	s = strings.TrimSuffix(s, "USDT")
	return strings.Trim(s, "_")
}

// MatchCloseQuantity accepts tiny float noise between UI observation and venue size.
func MatchCloseQuantity(requested, actual float64) bool {
	if requested <= 0 || actual <= 0 {
		return false
	}
	if actual+1e-12 < requested {
		return false
	}
	// Reject growth beyond observation (new size added).
	if actual > requested*(1+1e-6)+1e-8 {
		return false
	}
	return true
}

func MatchCloseEntry(requested, actual float64) bool {
	if requested <= 0 || actual <= 0 {
		return false
	}
	if requested == actual {
		return true
	}
	rel := math.Abs(requested-actual) / requested
	return rel <= 1e-8 || math.Abs(requested-actual) <= 1e-8
}
