package api

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"strand/internal/market"
	"strand/internal/provider/hyperliquid"
	"strand/internal/store"

	"github.com/gin-gonic/gin"
)

type backtestPoint struct {
	Time   int64   `json:"time"`
	Equity float64 `json:"equity"`
	Close  float64 `json:"close"`
	Signal string  `json:"signal,omitempty"`
}

func (s *Service) consoleStrategyBacktest(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := strings.TrimSpace(c.Param("id"))
	strat, err := st.Strategy().Get(userID, id)
	if err != nil || strat == nil {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}

	var body struct {
		Symbol         string  `json:"symbol"`
		Exchange       string  `json:"exchange"`
		Interval       string  `json:"interval"`
		LookbackBars   int     `json:"lookback_bars"`
		InitialBalance float64 `json:"initial_balance"`
	}
	_ = c.ShouldBindJSON(&body)
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "BTC"
	}
	exchange := strings.ToLower(strings.TrimSpace(body.Exchange))
	if exchange == "" {
		exchange = "hyperliquid"
	}
	interval := strings.TrimSpace(body.Interval)
	parsed, parseErr := strat.ParseConfig()
	if parseErr != nil || parsed == nil {
		parsed = &store.StrategyConfig{}
	}
	cfg := *parsed
	if interval == "" {
		if cfg.Indicators.Klines.PrimaryTimeframe != "" {
			interval = cfg.Indicators.Klines.PrimaryTimeframe
		} else {
			interval = "1h"
		}
	}
	lookback := body.LookbackBars
	if lookback <= 20 {
		if cfg.Indicators.Klines.PrimaryCount > 20 {
			lookback = cfg.Indicators.Klines.PrimaryCount
		} else {
			lookback = 180
		}
	}
	if lookback > 500 {
		lookback = 500
	}
	initial := body.InitialBalance
	if initial <= 0 {
		initial = 1000
	}

	klines, err := s.fetchBacktestKlines(exchange, symbol, interval, lookback)
	if err != nil || len(klines) < 40 {
		writeJSON(c, http.StatusBadRequest, gin.H{
			"error": "not enough kline data for backtest",
			"hint":  errString(err),
		})
		return
	}

	// Strategy-aware sizing + hard gates (still a signal smoke, not full Autopilot replay).
	posRatio := cfg.RiskControl.BTCETHMaxPositionValueRatio
	if !isMajorBacktestSymbol(symbol) {
		posRatio = cfg.RiskControl.AltcoinMaxPositionValueRatio
	}
	if posRatio <= 0 {
		posRatio = 0.5
	}
	if posRatio > 1 {
		posRatio = 1
	}
	dailyLimit := cfg.RiskControl.DailyLossLimitPct
	maxDD := cfg.RiskControl.MaxDrawdownPct
	emaFast, emaSlow := backtestEMAPeriods(cfg.Indicators)

	cash := initial
	position := 0.0
	entry := 0.0
	peak := initial
	dayStart := initial
	dayAnchor := int64(0)
	curve := make([]backtestPoint, 0, len(klines))
	trades := 0
	wins := 0
	gateHits := make([]string, 0)
	stopped := false
	method := "ema_cross_long_only"
	if cfg.Indicators.EnableRSI {
		method = "ema_cross_rsi_filter_long_only"
	}

	for i, k := range klines {
		closePx := k.Close
		if closePx <= 0 {
			continue
		}
		dayKey := k.Time / 86400000
		if dayAnchor == 0 || dayKey != dayAnchor {
			dayAnchor = dayKey
			dayStart = cash
			if position > 0 {
				dayStart += position * closePx
			}
		}

		mark := cash
		if position > 0 {
			mark += position * closePx
		}
		if mark > peak {
			peak = mark
		}

		signal := ""
		if !stopped && i >= emaSlow-1 {
			fast := smaAt(klines, i, emaFast)
			slow := smaAt(klines, i, emaSlow)
			rsiOK := true
			if cfg.Indicators.EnableRSI {
				period := 14
				if len(cfg.Indicators.RSIPeriods) > 0 && cfg.Indicators.RSIPeriods[0] > 1 {
					period = cfg.Indicators.RSIPeriods[0]
				}
				rsi := rsiAt(klines, i, period)
				rsiOK = rsi > 0 && rsi < 70
			}

			if position == 0 && fast > slow && rsiOK {
				notional := mark * posRatio
				if notional < cfg.RiskControl.MinPositionSize && cfg.RiskControl.MinPositionSize > 0 {
					notional = math.Min(cfg.RiskControl.MinPositionSize, mark*0.95)
				}
				if notional > 0 && notional <= cash {
					position = notional / closePx
					entry = closePx
					cash -= notional
					trades++
					signal = "enter_long"
				}
			} else if position > 0 && fast < slow {
				proceeds := position * closePx
				if proceeds > entry*position {
					wins++
				}
				cash += proceeds
				position = 0
				entry = 0
				signal = "exit_cross"
			}
		}

		mark = cash
		if position > 0 {
			mark += position * closePx
		}
		if mark > peak {
			peak = mark
		}

		// Hard gates: flatten and stop new risk (mirrors Autopilot pause semantics).
		if !stopped {
			dayPnLPct := 0.0
			if dayStart > 0 {
				dayPnLPct = (mark - dayStart) / dayStart * 100
			}
			ddPct := 0.0
			if peak > 0 {
				ddPct = (peak - mark) / peak * 100
			}
			hitDaily := dailyLimit > 0 && dayPnLPct <= -dailyLimit
			hitDD := maxDD > 0 && ddPct >= maxDD
			if (hitDaily || hitDD) && position > 0 {
				cash += position * closePx
				if position*closePx > entry*position {
					wins++
				}
				position = 0
				entry = 0
				signal = "exit_hard_gate"
			}
			if hitDaily || hitDD {
				stopped = true
				if hitDaily {
					gateHits = append(gateHits, "daily_loss")
					signal = firstNonEmptyBT(signal, "halt_daily_loss")
				}
				if hitDD {
					gateHits = append(gateHits, "max_drawdown")
					signal = firstNonEmptyBT(signal, "halt_max_drawdown")
				}
			}
		}

		mark = cash
		if position > 0 {
			mark += position * closePx
		}
		curve = append(curve, backtestPoint{
			Time:   k.Time,
			Equity: round2(mark),
			Close:  closePx,
			Signal: signal,
		})
	}

	if position > 0 && len(klines) > 0 {
		last := klines[len(klines)-1].Close
		cash += position * last
		position = 0
		if len(curve) > 0 {
			curve[len(curve)-1].Equity = round2(cash)
			curve[len(curve)-1].Signal = firstNonEmptyBT(curve[len(curve)-1].Signal, "flatten_end")
		}
	}

	finalEq := cash
	if len(curve) > 0 {
		finalEq = curve[len(curve)-1].Equity
	}
	pnl := finalEq - initial
	pnlPct := 0.0
	if initial > 0 {
		pnlPct = pnl / initial * 100
	}
	winRate := 0.0
	if trades > 0 {
		winRate = float64(wins) / float64(trades) * 100
	}
	maxDDSeen := 0.0
	runPeak := initial
	for _, p := range curve {
		if p.Equity > runPeak {
			runPeak = p.Equity
		}
		if runPeak > 0 {
			dd := (runPeak - p.Equity) / runPeak * 100
			if dd > maxDDSeen {
				maxDDSeen = dd
			}
		}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"strategy_id":      strat.ID,
		"strategy_name":    strat.Name,
		"symbol":           symbol,
		"exchange":         exchange,
		"interval":         interval,
		"lookback_bars":    lookback,
		"initial_balance":  initial,
		"final_equity":     round2(finalEq),
		"total_pnl":        round2(pnl),
		"total_pnl_pct":    round2(pnlPct),
		"trades":           trades,
		"win_rate_pct":     round2(winRate),
		"max_drawdown_pct": round2(maxDDSeen),
		"position_ratio":   round2(posRatio),
		"ema_fast":         emaFast,
		"ema_slow":         emaSlow,
		"hard_gates": gin.H{
			"daily_loss_limit_pct": dailyLimit,
			"max_drawdown_pct":     maxDD,
			"hits":                 uniqueStrings(gateHits),
			"stopped":              stopped,
		},
		"method":       method,
		"note":         "Strategy-aware signal backtest using EMA cross (+ optional RSI filter) and RiskControl hard gates. Not a full Autopilot replay.",
		"equity_curve": curve,
		"ran_at":       time.Now().UTC(),
	})
}

func backtestEMAPeriods(ind store.IndicatorConfig) (fast, slow int) {
	fast, slow = 20, 50
	if len(ind.EMAPeriods) >= 2 {
		a, b := ind.EMAPeriods[0], ind.EMAPeriods[1]
		if a > 0 && b > 0 {
			if a < b {
				return a, b
			}
			return b, a
		}
	}
	if len(ind.EMAPeriods) == 1 && ind.EMAPeriods[0] > 1 {
		fast = ind.EMAPeriods[0]
		slow = fast * 2
	}
	return fast, slow
}

func isMajorBacktestSymbol(symbol string) bool {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	s = strings.TrimSuffix(s, "USDT")
	s = strings.TrimSuffix(s, "-USDT-SWAP")
	switch s {
	case "BTC", "ETH":
		return true
	default:
		return false
	}
}

func smaAt(klines []btKline, i, period int) float64 {
	if period <= 0 || i+1 < period {
		return 0
	}
	sum := 0.0
	for j := i - period + 1; j <= i; j++ {
		sum += klines[j].Close
	}
	return sum / float64(period)
}

func rsiAt(klines []btKline, i, period int) float64 {
	if period <= 0 || i < period {
		return 0
	}
	gain, loss := 0.0, 0.0
	for j := i - period + 1; j <= i; j++ {
		d := klines[j].Close - klines[j-1].Close
		if d >= 0 {
			gain += d
		} else {
			loss -= d
		}
	}
	if loss == 0 {
		return 100
	}
	rs := (gain / float64(period)) / (loss / float64(period))
	return 100 - (100 / (1 + rs))
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func firstNonEmptyBT(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type btKline struct {
	Time  int64
	Close float64
}

func (s *Service) fetchBacktestKlines(exchange, symbol, interval string, limit int) ([]btKline, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	exchange = normalizeExchangeQuery(exchange)
	out := make([]btKline, 0, limit)
	switch exchange {
	case "hyperliquid", "lighter", "aster":
		client := hyperliquid.NewClient()
		candles, err := client.GetCandles(ctx, symbol, hyperliquid.MapTimeframe(interval), limit)
		if err != nil {
			return nil, err
		}
		for _, candle := range candles {
			closePx, _ := strconv.ParseFloat(candle.Close, 64)
			out = append(out, btKline{Time: candle.OpenTime, Close: closePx})
		}
	default:
		apiClient := market.NewAPIClient()
		klines, err := apiClient.GetKlines(symbol, interval, limit)
		if err != nil {
			return nil, err
		}
		for _, k := range klines {
			out = append(out, btKline{Time: k.OpenTime, Close: k.Close})
		}
	}
	return out, nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
