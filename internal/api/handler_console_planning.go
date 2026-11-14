package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"strand/internal/market"
	"strand/internal/planning/core"
	"strand/internal/provider/hyperliquid"
	"strand/internal/store"
	"strand/internal/trader/types"

	"github.com/gin-gonic/gin"
)

// Live order submission (approve/execute/stop/…) is not wired in this build.
// Readonly + compile endpoints keep Trade usable for planning without faking fills.
var errPlanningLiveUnavailable = errors.New(
	"Live order submission is not available in this build. Plans can be compiled and reviewed; Autopilot traders still place exchange orders.",
)

var (
	planningPrefsMu sync.RWMutex
	planningPrefs   = map[string]string{}

	planningStateMu sync.RWMutex
	planningStates  = map[string]*planningUserState{}
)

type planningUserState struct {
	Records map[string]planningRecord `json:"records"`
	Runs    []gin.H                   `json:"runs"`
}

type planningRecord struct {
	Plan  core.Plan `json:"plan"`
	State string    `json:"state"`
}

func (s *Service) userPlanningState(userID string) *planningUserState {
	planningStateMu.Lock()
	defer planningStateMu.Unlock()
	st, ok := planningStates[userID]
	if !ok || st == nil {
		st = &planningUserState{Records: map[string]planningRecord{}, Runs: []gin.H{}}
		planningStates[userID] = st
	}
	if st.Records == nil {
		st.Records = map[string]planningRecord{}
	}
	return st
}

func (s *Service) consolePlanningState(c *gin.Context) {
	userID := currentUserID(c)
	st := s.userPlanningState(userID)
	planningStateMu.RLock()
	records := make(map[string]planningRecord, len(st.Records))
	for k, v := range st.Records {
		records[k] = v
	}
	runs := append([]gin.H{}, st.Runs...)
	planningStateMu.RUnlock()
	writeJSON(c, http.StatusOK, gin.H{
		"records":  records,
		"accounts": map[string]string{},
		"runs":     runs,
		"mode":     "live",
		"now":      time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) consolePlanningConversations(c *gin.Context) {
	writeJSON(c, http.StatusOK, []any{})
}

func (s *Service) consolePlanningPreferencesGet(c *gin.Context) {
	userID := currentUserID(c)
	planningPrefsMu.RLock()
	text := planningPrefs[userID]
	planningPrefsMu.RUnlock()
	writeJSON(c, http.StatusOK, gin.H{"text": text})
}

func (s *Service) consolePlanningPreferencesPut(c *gin.Context) {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if len(body.Text) > 1600 {
		writeErr(c, http.StatusBadRequest, errors.New("preferences exceed 1600 characters"))
		return
	}
	userID := currentUserID(c)
	planningPrefsMu.Lock()
	planningPrefs[userID] = body.Text
	planningPrefsMu.Unlock()
	writeJSON(c, http.StatusOK, gin.H{"text": body.Text})
}

func (s *Service) consolePlanningSetupWallet(c *gin.Context) {
	if c.Request.Method == http.MethodPost {
		writeErr(c, http.StatusNotImplemented, errors.New(
			"Claw402 fee wallet setup is not available in this build. Use a direct AI provider key in Settings, or Autopilot traders for live execution.",
		))
		return
	}
	// Soft empty wallet: FrontierSetup treats found=false as "missing" without 404.
	writeJSON(c, http.StatusOK, gin.H{
		"found":          false,
		"enabled":        false,
		"chain":          "base",
		"asset":          "USDC",
		"balance_status": "unavailable",
	})
}

func (s *Service) consolePlanningMarkets(c *gin.Context) {
	venue := normalizeExchangeQuery(c.DefaultQuery("venue", "binance"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 35*time.Second)
	defer cancel()

	markets := make([]gin.H, 0)
	sources := []string{}
	warnings := []string{}

	switch venue {
	case "hyperliquid":
		coins, err := hyperliquid.GetProvider().GetAllCoins(ctx)
		if err != nil {
			writeErr(c, http.StatusBadGateway, err)
			return
		}
		sources = append(sources, "hyperliquid")
		for _, coin := range coins {
			base := coin.Symbol
			markets = append(markets, gin.H{
				"symbol":              coin.Symbol,
				"base":                base,
				"category":            coinCategory(coin.Symbol),
				"volume_24h":          coin.Volume24h,
				"price":               formatPlanningDecimal(coin.MarkPrice),
				"execution_supported": true,
			})
			if len(markets) >= 400 {
				break
			}
		}
	case "binance", "okx", "bitget", "bybit", "gate", "kucoin":
		if market.SupportsLiveCEXBoard(venue) {
			live, err := market.FetchCEXBoard(ctx, venue, 400)
			if err != nil {
				market.LogCEXBoardError(venue, err)
				warnings = append(warnings, err.Error())
			} else {
				sources = append(sources, "live")
				execOK := venue == "binance" || venue == "okx" || venue == "bitget"
				for _, coin := range live {
					markets = append(markets, gin.H{
						"symbol":              coin.Symbol,
						"base":                coin.Display,
						"category":            coin.Category,
						"volume_24h":          coin.Volume24h,
						"price":               formatPlanningDecimal(coin.MarkPrice),
						"execution_supported": execOK,
					})
				}
			}
		}
		if len(markets) == 0 {
			sources = append(sources, "curated")
			for _, item := range curatedCEXSymbols {
				sym, _ := item["symbol"].(string)
				display, _ := item["display"].(string)
				cat, _ := item["category"].(string)
				if display == "" {
					display = strings.TrimSuffix(sym, "USDT")
				}
				outSym := sym
				if venue == "okx" {
					outSym = display + "-USDT-SWAP"
				}
				markets = append(markets, gin.H{
					"symbol":              outSym,
					"base":                display,
					"category":            cat,
					"volume_24h":          0.0,
					"price":               "0",
					"execution_supported": venue == "binance" || venue == "okx" || venue == "bitget",
				})
			}
		}
	default:
		warnings = append(warnings, "unsupported venue")
	}

	writeJSON(c, http.StatusOK, gin.H{
		"venue":      venue,
		"markets":    markets,
		"fetched_at": time.Now().UTC().Format(time.RFC3339Nano),
		"sources":    sources,
		"warnings":   warnings,
	})
}

func (s *Service) consolePlanningAI500(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{
		"coins":      []any{},
		"source":     "unavailable",
		"fetched_at": time.Now().UTC().Format(time.RFC3339Nano),
		"message":    "AI500 paid data is not wired in this build",
	})
}

func (s *Service) consolePlanningAccount(c *gin.Context) {
	exchangeID := strings.TrimSpace(c.Query("exchange_id"))
	symbol := strings.TrimSpace(c.Query("symbol"))
	if exchangeID == "" {
		writeErr(c, http.StatusBadRequest, errors.New("exchange_id required"))
		return
	}
	userID := currentUserID(c)
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	ex, err := st.Exchange().GetByID(userID, exchangeID)
	if err != nil || ex == nil {
		writeErr(c, http.StatusNotFound, errors.New("exchange account not found"))
		return
	}
	tr, err := buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	bal, err := tr.GetBalance()
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	equity, _ := floatFromBalanceMap(bal, "totalEquity", "total_equity", "totalWalletBalance")
	avail, _ := floatFromBalanceMap(bal, "availableBalance", "available_balance")
	openOrders := 0
	if symbol != "" {
		if orders, oerr := tr.GetOpenOrders(symbol); oerr == nil {
			openOrders = len(orders)
		}
	}
	positionQty := 0.0
	if positions, perr := tr.GetPositions(); perr == nil {
		for _, p := range positions {
			sym, _ := p["symbol"].(string)
			if symbol != "" && !strings.EqualFold(sym, symbol) {
				continue
			}
			if q, ok := floatFromBalanceMap(p, "quantity", "positionAmt", "size"); ok {
				positionQty += math.Abs(q)
			}
		}
	}
	writeJSON(c, http.StatusOK, gin.H{
		"equity":      formatPlanningDecimal(equity),
		"available":   formatPlanningDecimal(avail),
		"position":    formatPlanningDecimal(positionQty),
		"open_orders": openOrders,
	})
}

func (s *Service) consoleOpenOrders(c *gin.Context) {
	userID := currentUserID(c)
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	symbol := strings.TrimSpace(c.Query("symbol"))
	exchangeID := strings.TrimSpace(c.Query("exchange_id"))
	traderID := strings.TrimSpace(c.Query("trader_id"))

	var ex *store.Exchange
	var err error
	switch {
	case exchangeID != "":
		ex, err = st.Exchange().GetByID(userID, exchangeID)
		if err != nil || ex == nil {
			writeErr(c, http.StatusNotFound, errors.New("exchange account not found"))
			return
		}
	case traderID != "":
		tr, terr := st.Trader().GetByID(traderID)
		if terr != nil || tr == nil || tr.UserID != userID {
			writeErr(c, http.StatusNotFound, errors.New("trader not found"))
			return
		}
		if tr.ExchangeID == "" {
			writeJSON(c, http.StatusOK, []any{})
			return
		}
		ex, err = st.Exchange().GetByID(userID, tr.ExchangeID)
		if err != nil || ex == nil {
			writeJSON(c, http.StatusOK, []any{})
			return
		}
	default:
		writeErr(c, http.StatusBadRequest, errors.New("exchange_id or trader_id required"))
		return
	}

	trader, err := buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	orders, err := trader.GetOpenOrders(symbol)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	out := make([]gin.H, 0, len(orders))
	for _, o := range orders {
		out = append(out, openOrderJSON(o))
	}
	writeJSON(c, http.StatusOK, out)
}

func openOrderJSON(o types.OpenOrder) gin.H {
	return gin.H{
		"order_id":      o.OrderID,
		"symbol":        o.Symbol,
		"side":          o.Side,
		"position_side": o.PositionSide,
		"type":          o.Type,
		"price":         o.Price,
		"stop_price":    o.StopPrice,
		"quantity":      o.Quantity,
		"status":        o.Status,
	}
}

func (s *Service) consolePlanningCompile(c *gin.Context) {
	var body struct {
		ExchangeID  string           `json:"exchange_id"`
		Venue       string           `json:"venue"`
		Symbol      string           `json:"symbol"`
		Idea        core.Idea        `json:"idea"`
		MarketGroup string           `json:"market_group"`
		Constraints core.Constraints `json:"constraints"`
		AutoSize    bool             `json:"auto_size"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	venue := normalizeExchangeQuery(body.Venue)
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		writeErr(c, http.StatusBadRequest, errors.New("symbol required"))
		return
	}
	switch venue {
	case "binance", "okx", "bitget", "hyperliquid":
	default:
		writeErr(c, http.StatusBadRequest, fmt.Errorf("planning compile supports binance, okx, bitget, hyperliquid; got %s", venue))
		return
	}

	userID := currentUserID(c)
	now := time.Now().UTC()
	constraints := body.Constraints
	if constraints.FeeBPS == "" {
		constraints.FeeBPS = "10"
	}
	if constraints.SlippageBPS == "" {
		constraints.SlippageBPS = "10"
	}
	if constraints.MinRewardRisk == "" {
		constraints.MinRewardRisk = "0"
	}
	if body.MarketGroup == "ai500" && constraints.EntryPolicy == "" {
		constraints.EntryPolicy = core.EntryAI500TrendLong
	}

	var trader types.Trader
	if body.ExchangeID != "" {
		st := s.eng.Store()
		if st == nil {
			writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
			return
		}
		ex, err := st.Exchange().GetByID(userID, body.ExchangeID)
		if err != nil || ex == nil {
			writeErr(c, http.StatusNotFound, errors.New("exchange account not found"))
			return
		}
		trader, err = buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
		if err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
		if body.AutoSize {
			bal, berr := trader.GetBalance()
			if berr != nil {
				writeErr(c, http.StatusBadGateway, berr)
				return
			}
			equity, _ := floatFromBalanceMap(bal, "totalEquity", "total_equity", "totalWalletBalance")
			avail, _ := floatFromBalanceMap(bal, "availableBalance", "available_balance")
			if equity <= 0 {
				writeErr(c, http.StatusBadRequest, errors.New("account equity is zero; fund the exchange or explore without an account"))
				return
			}
			if avail < 0 {
				avail = 0
			}
			constraints.Equity = formatPlanningDecimal(equity)
			constraints.SizingPolicy = core.SizingAccountMargin
			lev := constraints.EffectiveLeverage()
			// Default risk budget: 1% of equity, notional up to available*leverage.
			risk := equity * 0.01
			if risk <= 0 {
				risk = 1
			}
			constraints.MaxRisk = formatPlanningDecimal(risk)
			maxNotional := avail * float64(lev)
			if maxNotional <= 0 {
				maxNotional = equity * 0.05
			}
			if maxNotional <= 0 {
				maxNotional = 100
			}
			constraints.MaxNotional = formatPlanningDecimal(maxNotional)
			if avail > 0 {
				constraints.AvailableMargin = formatPlanningDecimal(avail)
				constraints.PlanningOnly = false
			} else {
				// Verified empty margin → equity preview only (no live size claim).
				constraints.AvailableMargin = "0"
				constraints.PlanningOnly = true
			}
		}
	} else {
		if constraints.Equity == "" {
			constraints.Equity = "1000"
		}
		if constraints.MaxRisk == "" {
			constraints.MaxRisk = "10"
		}
		if constraints.MaxNotional == "" {
			constraints.MaxNotional = "100"
		}
		constraints.PlanningOnly = true
		constraints.SizingPolicy = core.SizingAccountMargin
		constraints.AvailableMargin = "0"
	}

	snap, err := s.buildPlanningSnapshot(c.Request.Context(), trader, venue, symbol, now)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	plan, err := core.Compile(snap, constraints, body.Idea, now)
	if err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}

	st := s.userPlanningState(userID)
	planningStateMu.Lock()
	st.Records[plan.ID] = planningRecord{Plan: plan, State: "draft"}
	planningStateMu.Unlock()

	writeJSON(c, http.StatusOK, plan)
}

func (s *Service) buildPlanningSnapshot(ctx context.Context, trader types.Trader, venue, symbol string, now time.Time) (core.Snapshot, error) {
	price := 0.0
	var perr error
	if trader != nil {
		price, perr = trader.GetMarketPrice(symbol)
	}
	if price <= 0 {
		price, perr = s.publicMarkPrice(ctx, venue, symbol)
	}
	if price <= 0 {
		if perr != nil {
			return core.Snapshot{}, perr
		}
		return core.Snapshot{}, errors.New("could not resolve market price for snapshot")
	}

	tick, step, minQty, minNotional := inferInstrumentRules(price)
	// Synthetic top-of-book within a tiny spread so Compile entry checks pass.
	half := price * 0.00005
	if half < mustFloat(tick) {
		half = mustFloat(tick)
	}
	bid := floorToTick(price-half, tick)
	ask := floorToTick(price+half, tick)
	if ask <= bid {
		ask = floorToTick(bid+mustFloat(tick), tick)
	}
	size := math.Max(minQty*1000, 10)

	base, quote := splitSymbolAssets(venue, symbol)
	return core.Snapshot{
		Venue:       venue,
		Symbol:      symbol,
		Kind:        "perpetual",
		BaseAsset:   base,
		QuoteAsset:  quote,
		Bid:         formatPlanningDecimal(bid),
		Ask:         formatPlanningDecimal(ask),
		BidSize:     formatPlanningDecimal(size),
		AskSize:     formatPlanningDecimal(size),
		Tick:        tick,
		Step:        step,
		MinQty:      formatPlanningDecimal(minQty),
		MinNotional: formatPlanningDecimal(minNotional),
		Observed:    now,
		SourceTime:  now,
	}, nil
}

func (s *Service) publicMarkPrice(ctx context.Context, venue, symbol string) (float64, error) {
	switch venue {
	case "hyperliquid":
		coins, err := hyperliquid.GetProvider().GetAllCoins(ctx)
		if err != nil {
			return 0, err
		}
		for _, coin := range coins {
			if strings.EqualFold(coin.Symbol, symbol) {
				return coin.MarkPrice, nil
			}
		}
		return 0, fmt.Errorf("symbol %s not found on hyperliquid", symbol)
	case "binance", "okx", "bitget", "bybit", "gate", "kucoin":
		if market.SupportsLiveCEXBoard(venue) {
			rows, err := market.FetchCEXBoard(ctx, venue, 500)
			if err == nil {
				for _, row := range rows {
					if strings.EqualFold(row.Symbol, symbol) || strings.EqualFold(row.Display, symbol) {
						if row.MarkPrice > 0 {
							return row.MarkPrice, nil
						}
					}
				}
			}
		}
		klines, err := market.FetchCEXKlines(ctx, venue, symbol, "1m", 2)
		if err != nil {
			return 0, err
		}
		if len(klines) == 0 || klines[len(klines)-1].Close <= 0 {
			return 0, fmt.Errorf("no price for %s on %s", symbol, venue)
		}
		return klines[len(klines)-1].Close, nil
	default:
		return 0, fmt.Errorf("unsupported venue %s", venue)
	}
}

func inferInstrumentRules(price float64) (tick, step core.Decimal, minQty, minNotional float64) {
	switch {
	case price >= 10000:
		return "0.1", "0.001", 0.001, 5
	case price >= 1000:
		return "0.01", "0.01", 0.01, 5
	case price >= 100:
		return "0.01", "0.1", 0.1, 5
	case price >= 1:
		return "0.0001", "1", 1, 5
	default:
		return "0.00001", "1", 1, 5
	}
}

func splitSymbolAssets(venue, symbol string) (base, quote string) {
	quote = "USDT"
	if venue == "hyperliquid" {
		quote = "USDC"
		return symbol, quote
	}
	s := strings.ToUpper(symbol)
	s = strings.ReplaceAll(s, "-USDT-SWAP", "")
	s = strings.ReplaceAll(s, "_USDT", "")
	s = strings.TrimSuffix(s, "USDT")
	s = strings.Trim(s, "-_")
	if s == "" {
		s = symbol
	}
	return s, quote
}

func formatPlanningDecimal(v float64) core.Decimal {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return core.Decimal(strconv.FormatFloat(v, 'f', -1, 64))
}

func mustFloat(d core.Decimal) float64 {
	f, _ := strconv.ParseFloat(string(d), 64)
	return f
}

func floorToTick(v float64, tick core.Decimal) float64 {
	t := mustFloat(tick)
	if t <= 0 {
		return v
	}
	return math.Floor(v/t+1e-12) * t
}

func (s *Service) consolePlanningLiveUnavailable(c *gin.Context) {
	writeErr(c, http.StatusNotImplemented, errPlanningLiveUnavailable)
}

func (s *Service) consolePlanningClosePosition(c *gin.Context) {
	writeJSON(c, http.StatusNotImplemented, gin.H{
		"error":       errPlanningLiveUnavailable.Error(),
		"close_state": "not_submitted",
	})
}

func (s *Service) consolePlanningChat(c *gin.Context) {
	writeErr(c, http.StatusNotImplemented, errors.New(
		"Planning chat is not available in this build. Use Custom market / limit / ladder / TWAP to compile a plan, or Chat for market Q&A.",
	))
}

func (s *Service) consolePlanningEntryPlugins(c *gin.Context) {
	writeJSON(c, http.StatusOK, []any{})
}

func (s *Service) consolePlanningDataPlugins(c *gin.Context) {
	writeJSON(c, http.StatusOK, []any{})
}

func (s *Service) consolePlanningPluginData(c *gin.Context) {
	writeErr(c, http.StatusNotImplemented, errors.New("planning data plugins are not available in this build"))
}

func (s *Service) consolePlanningInstrument(c *gin.Context) {
	venue := normalizeExchangeQuery(c.DefaultQuery("venue", "binance"))
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		writeErr(c, http.StatusBadRequest, errors.New("symbol required"))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	price, err := s.publicMarkPrice(ctx, venue, symbol)
	if err != nil || price <= 0 {
		price = 100
	}
	tick, step, minQty, minNotional := inferInstrumentRules(price)
	base, quote := splitSymbolAssets(venue, symbol)
	writeJSON(c, http.StatusOK, gin.H{
		"venue":        venue,
		"symbol":       symbol,
		"kind":         "perpetual",
		"base_asset":   base,
		"quote_asset":  quote,
		"tick":         tick,
		"step":         step,
		"min_qty":      formatPlanningDecimal(minQty),
		"min_notional": formatPlanningDecimal(minNotional),
	})
}

func (s *Service) consolePlanningCandles(c *gin.Context) {
	venue := normalizeExchangeQuery(c.DefaultQuery("venue", c.DefaultQuery("exchange", "binance")))
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		writeErr(c, http.StatusBadRequest, errors.New("symbol required"))
		return
	}
	interval := c.DefaultQuery("interval", "1h")
	limit := 300
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	candles := make([]gin.H, 0)
	switch venue {
	case "hyperliquid", "lighter", "aster":
		client := hyperliquid.NewClient()
		hlInterval := hyperliquid.MapTimeframe(interval)
		rows, err := client.GetCandles(ctx, symbol, hlInterval, limit)
		if err != nil {
			writeErr(c, http.StatusBadGateway, err)
			return
		}
		for _, candle := range rows {
			open, _ := strconv.ParseFloat(candle.Open, 64)
			high, _ := strconv.ParseFloat(candle.High, 64)
			low, _ := strconv.ParseFloat(candle.Low, 64)
			closePx, _ := strconv.ParseFloat(candle.Close, 64)
			vol, _ := strconv.ParseFloat(candle.Volume, 64)
			t := candle.OpenTime
			if t > 1e12 {
				t = t / 1000
			}
			candles = append(candles, gin.H{
				"time": t, "open": open, "high": high, "low": low, "close": closePx, "volume": vol,
			})
		}
	case "binance", "bybit", "okx", "gate", "bitget", "kucoin":
		rows, err := market.FetchCEXKlines(ctx, venue, symbol, interval, limit)
		if err != nil {
			market.LogCEXBoardError(venue+" planning candles", err)
			writeJSON(c, http.StatusOK, gin.H{"venue": venue, "symbol": symbol, "interval": interval, "candles": []any{}})
			return
		}
		for _, k := range rows {
			t := k.OpenTime
			if t > 1e12 {
				t = t / 1000
			}
			candles = append(candles, gin.H{
				"time": t, "open": k.Open, "high": k.High, "low": k.Low, "close": k.Close, "volume": k.Volume,
			})
		}
	}

	// Optional `to` cursor: keep bars strictly before to (unix seconds).
	if toRaw := c.Query("to"); toRaw != "" {
		if to, err := strconv.ParseInt(toRaw, 10, 64); err == nil && to > 0 {
			filtered := candles[:0]
			for _, row := range candles {
				if t, ok := row["time"].(int64); ok && t < to {
					filtered = append(filtered, row)
				} else if tf, ok := row["time"].(float64); ok && int64(tf) < to {
					filtered = append(filtered, row)
				}
			}
			candles = filtered
		}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"venue":    venue,
		"symbol":   symbol,
		"interval": interval,
		"candles":  candles,
	})
}

func (s *Service) consolePlanningFeed(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case <-time.After(15 * time.Second):
			_, _ = io.WriteString(w, ": keepalive\n\n")
			return true
		}
	})
}
