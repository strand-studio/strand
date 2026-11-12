package binance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strand/internal/logger"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

const futuresBrokerID = "KzrpZaP9"
const futuresBrokerPrefix = "x-" + futuresBrokerID

// getBrOrderID generates unique order ID (for futures contracts)
// Format: x-{BR_ID}{TIMESTAMP}{RANDOM}
// Futures limit is 32 characters, use this limit consistently
// Uses nanosecond timestamp + random number to ensure global uniqueness (collision probability < 10^-20)
func getBrOrderID() string {
	brID := futuresBrokerID

	timestamp := time.Now().UnixNano() % 10000000000000 // 13-digit nanosecond timestamp

	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)

	orderID := fmt.Sprintf("x-%s%d%s", brID, timestamp, randomHex)

	if len(orderID) > 32 {
		orderID = orderID[:32]
	}

	return orderID
}

// FuturesTrader Binance futures trader
type FuturesTrader struct {
	client *futures.Client

	// Balance cache
	cachedBalance     map[string]interface{}
	balanceCacheTime  time.Time
	balanceCacheMutex sync.RWMutex

	// Position cache
	cachedPositions     []map[string]interface{}
	positionsCacheTime  time.Time
	positionsCacheMutex sync.RWMutex

	// Cache validity period (15 seconds)
	cacheDuration time.Duration
}

// NewFuturesTrader creates futures trader
func NewFuturesTrader(apiKey, secretKey string, userId string) *FuturesTrader {
	trader := NewExecutionTrader(apiKey, secretKey, userId)
	syncBinanceServerTime(trader.client)

	// Set dual-side position mode (Hedge Mode)
	// This is required because the code uses PositionSide (LONG/SHORT)
	if err := trader.setDualSidePosition(); err != nil {
		logger.Infof("⚠️ Failed to set dual-side position mode: %v (ignore this warning if already in dual-side mode)", err)
	}

	return trader

	// return &FuturesTrader{
	// 	client:        futures.NewClient(apiKey, secretKey),
	// 	cacheDuration: 15 * time.Second,
	// }
}

// setDualSidePosition sets dual-side position mode (called during initialization)
func (t *FuturesTrader) setDualSidePosition() error {
	err := t.client.NewChangePositionModeService().
		DualSide(true).
		Do(context.Background())

	if err != nil {
		if strings.Contains(err.Error(), "No need to change position side") {
			logger.Infof("  ✓ Account is already in dual-side position mode (Hedge Mode)")
			return nil
		}
		return err
	}

	logger.Infof("  ✓ Account has been switched to dual-side position mode (Hedge Mode)")
	logger.Infof("  ℹ️  Dual-side position mode allows holding both long and short positions simultaneously")
	return nil
}

// syncBinanceServerTime syncs Binance server time to ensure request timestamps are valid
func syncBinanceServerTime(client *futures.Client) {
	serverTime, err := client.NewServerTimeService().Do(context.Background())
	if err != nil {
		logger.Infof("⚠️ Failed to sync Binance server time: %v", err)
		return
	}

	now := time.Now().UnixMilli()
	offset := now - serverTime
	client.TimeOffset = offset
	logger.Infof("⏱ Binance server time synced, offset %dms", offset)
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculatePrecision calculates precision from stepSize
func calculatePrecision(stepSize string) int {
	// Remove trailing zeros
	stepSize = trimTrailingZeros(stepSize)

	// Find decimal point
	dotIndex := -1
	for i := 0; i < len(stepSize); i++ {
		if stepSize[i] == '.' {
			dotIndex = i
			break
		}
	}

	// If no decimal point or decimal point is at the end, precision is 0
	if dotIndex == -1 || dotIndex == len(stepSize)-1 {
		return 0
	}

	// Return number of digits after decimal point
	return len(stepSize) - dotIndex - 1
}

// trimTrailingZeros removes trailing zeros
func trimTrailingZeros(s string) string {
	// If no decimal point, return directly
	if !stringContains(s, ".") {
		return s
	}

	// Iterate backwards to remove trailing zeros
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}

	// If last character is decimal point, remove it too
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}

	return s
}

// ----api----

// Position is a USDT-M futures position. Qty is the base-asset size.
type Position struct {
	Symbol   string
	Side     string
	Qty      float64
	Entry    float64
	Mark     float64
	Leverage int
	UPnL     float64
	Margin   float64
}

// SetLeverage sets the initial leverage for a symbol.
// func (t *FuturesTrader) SetLeverage(ctx context.Context, symbol string, leverage int) error {
// 	if leverage < 1 {
// 		leverage = 1
// 	}
// 	_, err := t.client.NewChangeLeverageService().Symbol(symbol).Leverage(leverage).Do(ctx)
// 	if err != nil {
// 		return fmt.Errorf("设置杠杆失败: %w", err)
// 	}
// 	return nil
// }

// MarketOrder sends a market order.
// positionSide is "long" or "short" — the position being opened or closed, not the order side.
// hedge must be true only when the account is in dual-side mode.
func (t *FuturesTrader) MarketOrder(ctx context.Context, symbol, positionSide string, qty float64, reduceOnly, hedge bool) (string, float64, error) {
	if qty <= 0 {
		return "", 0, fmt.Errorf("数量要大于 0")
	}
	svc := t.client.NewCreateOrderService().
		Symbol(symbol).
		Side(futuresSide(positionSide, reduceOnly)).
		Type(futures.OrderTypeMarket).
		Quantity(formatQty(qty))
	if reduceOnly {
		svc.ReduceOnly(true)
	}
	if hedge {
		if positionSide == "short" {
			svc.PositionSide(futures.PositionSideTypeShort)
		} else {
			svc.PositionSide(futures.PositionSideTypeLong)
		}
	}
	resp, err := svc.Do(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("交易所拒绝: %w", err)
	}
	return strconv.FormatInt(resp.OrderID, 10), atof(resp.AvgPrice), nil
}

// Account returns the USDT wallet and open positions.
func (t *FuturesTrader) Account(ctx context.Context) (float64, float64, []Position, error) {
	balances, err := t.client.NewGetBalanceService().Do(ctx)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("读取币安余额失败: %w", err)
	}
	var available, equity float64
	for _, b := range balances {
		if !strings.EqualFold(b.Asset, "USDT") {
			continue
		}
		available = atof(b.AvailableBalance)
		equity = atof(b.Balance) + atof(b.CrossUnPnl)
		break
	}

	risks, err := t.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return available, equity, nil, fmt.Errorf("读取币安持仓失败: %w", err)
	}
	out := make([]Position, 0)
	for _, p := range risks {
		amt := atof(p.PositionAmt)
		if math.Abs(amt) == 0 {
			continue
		}
		side := "long"
		qty := amt
		switch strings.ToUpper(p.PositionSide) {
		case "SHORT":
			side = "short"
		case "LONG":
			side = "long"
		default:
			if amt < 0 {
				side = "short"
			}
		}
		if qty < 0 {
			qty = -qty
		}
		lev, _ := strconv.Atoi(p.Leverage)
		notional := math.Abs(atof(p.Notional))
		margin := 0.0
		if lev > 0 {
			margin = notional / float64(lev)
		}
		if iso := atof(p.IsolatedWallet); iso > 0 {
			margin = iso
		}
		out = append(out, Position{
			Symbol:   p.Symbol,
			Side:     side,
			Qty:      qty,
			Entry:    atof(p.EntryPrice),
			Mark:     atof(p.MarkPrice),
			Leverage: lev,
			UPnL:     atof(p.UnRealizedProfit),
			Margin:   margin,
		})
	}
	return available, equity, out, nil
}

func futuresSide(positionSide string, reduceOnly bool) futures.SideType {
	buy := positionSide != "short"
	if reduceOnly {
		buy = !buy
	}
	if buy {
		return futures.SideTypeBuy
	}
	return futures.SideTypeSell
}

func formatQty(v float64) string {
	s := strconv.FormatFloat(v, 'f', 8, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
