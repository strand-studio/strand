package api

import (
	"context"
	"encoding/hex"
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
	"strand/internal/planning/live"
	"strand/internal/provider/hyperliquid"
	"strand/internal/store"
	"strand/internal/trader/types"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

var (
	planningPrefsMu sync.RWMutex
	planningPrefs   = map[string]string{}

	planningStateMu sync.RWMutex
	planningStates  = map[string]*planningUserState{}
)

type planningUserState struct {
	Records map[string]planningRecord `json:"records"`
}

type planningRecord struct {
	Plan       core.Plan `json:"plan"`
	State      string    `json:"state"`
	ExchangeID string    `json:"exchange_id,omitempty"`
}

func (s *Service) userPlanningState(userID string) *planningUserState {
	planningStateMu.Lock()
	defer planningStateMu.Unlock()
	st, ok := planningStates[userID]
	if !ok || st == nil {
		st = &planningUserState{Records: map[string]planningRecord{}}
		planningStates[userID] = st
	}
	if st.Records == nil {
		st.Records = map[string]planningRecord{}
	}
	return st
}

func (s *Service) planningLiveEngine() (*live.Engine, error) {
	st := s.eng.Store()
	if st == nil || st.GormDB() == nil {
		return nil, errStoreUnavailable
	}
	factory := func(ctx context.Context, userID, exchangeID string) (live.Broker, string, error) {
		_ = ctx
		return buildPlanningBroker(st, userID, exchangeID)
	}
	snap := func(ctx context.Context, userID, exchangeID, venue, symbol string) (core.Snapshot, error) {
		var trader types.Trader
		if exchangeID != "" {
			ex, err := st.Exchange().GetByID(userID, exchangeID)
			if err != nil || ex == nil {
				return core.Snapshot{}, errors.New("exchange account not found")
			}
			trader, err = buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
			if err != nil {
				return core.Snapshot{}, err
			}
		}
		return s.buildPlanningSnapshot(ctx, trader, venue, symbol, time.Now().UTC())
	}
	return live.NewEngine(st.GormDB(), factory, snap), nil
}

func liveRunJSON(v live.View) gin.H {
	return gin.H{
		"id":            v.ID,
		"exchange_id":   v.ExchangeID,
		"plan_id":       v.PlanID,
		"plan_hash":     v.PlanHash,
		"approval_hash": v.ApprovalHash,
		"updated_at":    v.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"state":         v.State,
		"error":         v.Error,
		"plan":          v.Plan,
		"orders":        v.Orders,
	}
}

func (s *Service) consolePlanningState(c *gin.Context) {
	userID := currentUserID(c)
	st := s.userPlanningState(userID)
	planningStateMu.RLock()
	records := make(map[string]planningRecord, len(st.Records))
	accounts := map[string]string{}
	for k, v := range st.Records {
		records[k] = v
		if v.ExchangeID != "" {
			accounts[v.Plan.ID] = v.ExchangeID
		}
	}
	planningStateMu.RUnlock()

	runs := []gin.H{}
	if eng, err := s.planningLiveEngine(); err == nil {
		if views, lerr := eng.ListViews(userID); lerr == nil {
			runs = make([]gin.H, 0, len(views))
			for _, v := range views {
				runs = append(runs, liveRunJSON(v))
				if v.PlanID != "" && v.ExchangeID != "" {
					accounts[v.PlanID] = v.ExchangeID
				}
			}
		}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"records":  records,
		"accounts": accounts,
		"runs":     runs,
		"mode":     "live",
		"now":      time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) consolePlanningConversations(c *gin.Context) {
	userID := currentUserID(c)
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	rows, err := st.PlanningConversations().List(userID, 40)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{
			"id":          row.ID,
			"symbol":      row.Symbol,
			"exchange_id": row.ExchangeID,
			"model_id":    row.ModelID,
			"messages":    row.Messages,
			"updated_at":  row.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(c, http.StatusOK, out)
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
	userID := currentUserID(c)
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}

	if c.Request.Method == http.MethodPost {
		var body struct {
			ModelID string `json:"model_id"`
		}
		_ = c.ShouldBindJSON(&body)
		modelID, address, _, enabled, found := s.findClaw402Model(userID)
		if found {
			if !enabled && modelID != "" {
				_ = st.AIModel().UpdateWithName(userID, modelID, "Claw402", true, "", "", beginnerDefaultModel)
			}
		} else {
			// Create a fee wallet the same way beginner onboarding does.
			pk, err := crypto.GenerateKey()
			if err != nil {
				writeErr(c, http.StatusInternalServerError, err)
				return
			}
			priv := "0x" + hex.EncodeToString(crypto.FromECDSA(pk))
			address = crypto.PubkeyToAddress(pk.PublicKey).Hex()
			if err := st.AIModel().UpdateWithName(
				userID, "claw402", "Claw402", true, priv, "", beginnerDefaultModel,
			); err != nil {
				writeErr(c, http.StatusInternalServerError, err)
				return
			}
			modelID, address, _, _, found = s.findClaw402Model(userID)
			if !found {
				writeErr(c, http.StatusInternalServerError, errors.New("failed to persist Claw402 fee wallet"))
				return
			}
		}
		_ = body
		bal, status := cachedBaseUSDCBalance(address)
		writeJSON(c, http.StatusOK, gin.H{
			"found":          true,
			"enabled":        true,
			"address":        address,
			"model_id":       modelID,
			"chain":          "base",
			"asset":          "USDC",
			"balance_usdc":   bal,
			"balance_status": status,
		})
		return
	}

	modelID, address, _, enabled, found := s.findClaw402Model(userID)
	if !found {
		writeJSON(c, http.StatusOK, gin.H{
			"found":          false,
			"enabled":        false,
			"chain":          "base",
			"asset":          "USDC",
			"balance_status": "unavailable",
		})
		return
	}
	bal, status := cachedBaseUSDCBalance(address)
	writeJSON(c, http.StatusOK, gin.H{
		"found":          true,
		"enabled":        enabled,
		"address":        address,
		"model_id":       modelID,
		"chain":          "base",
		"asset":          "USDC",
		"balance_usdc":   bal,
		"balance_status": status,
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	// Paid Claw402 AI500 feed is not wired; expose Binance volume board as a
	// usable long-candidate shortcut with an explicit proxy source.
	coins := make([]gin.H, 0)
	source := "binance_volume_proxy"
	message := "Claw402 AI500 paid feed is not wired; showing Binance USDT perpetual volume leaders as a temporary shortcut."
	if market.SupportsLiveCEXBoard("binance") {
		live, err := market.FetchCEXBoard(ctx, "binance", 80)
		if err != nil {
			market.LogCEXBoardError("binance ai500 proxy", err)
			message = err.Error()
			source = "unavailable"
		} else {
			for _, coin := range live {
				coins = append(coins, gin.H{
					"pair":  coin.Symbol,
					"score": coin.Volume24h,
				})
			}
		}
	}
	writeJSON(c, http.StatusOK, gin.H{
		"coins":      coins,
		"source":     source,
		"fetched_at": time.Now().UTC().Format(time.RFC3339Nano),
		"message":    message,
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
	st.Records[plan.ID] = planningRecord{
		Plan:       plan,
		State:      "draft",
		ExchangeID: strings.TrimSpace(body.ExchangeID),
	}
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

	instCtx, cancel := planningInstrumentHTTPTimeout(ctx)
	defer cancel()
	rules := s.resolvePlanningInstrument(instCtx, venue, symbol, price)
	tick, step, minQty, minNotional := rules.Tick, rules.Step, rules.MinQty, rules.MinNotional
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
		Venue:         venue,
		Symbol:        symbol,
		Kind:          "perpetual",
		BaseAsset:     base,
		QuoteAsset:    quote,
		ContractValue: rules.ContractValue,
		Bid:           formatPlanningDecimal(bid),
		Ask:           formatPlanningDecimal(ask),
		BidSize:       formatPlanningDecimal(size),
		AskSize:       formatPlanningDecimal(size),
		Tick:          tick,
		Step:          step,
		MinQty:        formatPlanningDecimal(minQty),
		MinNotional:   formatPlanningDecimal(minNotional),
		Observed:      now,
		SourceTime:    now,
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

var errAutopilotRunning = errors.New("Stop Autopilot for this account in Portfolio before manually trading.")

func (s *Service) assertNoRunningAutopilot(userID, exchangeID string) error {
	exchangeID = strings.TrimSpace(exchangeID)
	if exchangeID == "" {
		return nil
	}
	st := s.eng.Store()
	if st == nil {
		return errStoreUnavailable
	}
	traders, err := st.Trader().ListByExchangeID(userID, exchangeID)
	if err != nil {
		return err
	}
	for _, tr := range traders {
		if tr != nil && tr.IsRunning {
			return errAutopilotRunning
		}
	}
	return nil
}

func (s *Service) consolePlanningApprove(c *gin.Context) {
	var body struct {
		PlanID              string `json:"plan_id"`
		Hash                string `json:"hash"`
		Confirmed           bool   `json:"confirmed"`
		ExchangeID          string `json:"exchange_id"`
		ReplaceApprovalID   string `json:"replace_approval_id"`
		ReplaceApprovalHash string `json:"replace_approval_hash"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	userID := currentUserID(c)
	st := s.userPlanningState(userID)
	planningStateMu.RLock()
	rec, ok := st.Records[body.PlanID]
	planningStateMu.RUnlock()
	if !ok {
		writeErr(c, http.StatusNotFound, errors.New("plan draft not found; compile again"))
		return
	}
	exchangeID := strings.TrimSpace(body.ExchangeID)
	if exchangeID == "" {
		exchangeID = strings.TrimSpace(rec.ExchangeID)
	}
	if exchangeID != "" {
		if err := s.assertNoRunningAutopilot(userID, exchangeID); err != nil {
			writeErr(c, http.StatusConflict, err)
			return
		}
	}
	eng, err := s.planningLiveEngine()
	if err != nil {
		writeErr(c, http.StatusServiceUnavailable, err)
		return
	}
	view, err := eng.Approve(c.Request.Context(), live.ApproveInput{
		UserID:              userID,
		ExchangeID:          exchangeID,
		Plan:                rec.Plan,
		Hash:                body.Hash,
		Confirmed:           body.Confirmed,
		ReplaceApprovalID:   body.ReplaceApprovalID,
		ReplaceApprovalHash: body.ReplaceApprovalHash,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrUnresolvedPlan) || errors.Is(err, live.ErrAccountLease) {
			status = http.StatusConflict
		}
		writeErr(c, status, err)
		return
	}
	planningStateMu.Lock()
	rec.State = "approved"
	rec.ExchangeID = exchangeID
	st.Records[body.PlanID] = rec
	planningStateMu.Unlock()
	writeJSON(c, http.StatusOK, liveRunJSON(view))
}

func (s *Service) consolePlanningExecute(c *gin.Context) {
	var body struct {
		ID   string `json:"id"`
		Hash string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	userID := currentUserID(c)
	eng, err := s.planningLiveEngine()
	if err != nil {
		writeErr(c, http.StatusServiceUnavailable, err)
		return
	}
	if views, lerr := eng.ListViews(userID); lerr == nil {
		for _, v := range views {
			if v.ID == body.ID && v.ExchangeID != "" {
				if err := s.assertNoRunningAutopilot(userID, v.ExchangeID); err != nil {
					writeErr(c, http.StatusConflict, err)
					return
				}
				break
			}
		}
	}
	view, err := eng.Execute(c.Request.Context(), live.ExecuteInput{
		UserID:       userID,
		RunID:        body.ID,
		ApprovalHash: body.Hash,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrNotFound) {
			status = http.StatusNotFound
		}
		// Still return body when partial progress was persisted.
		if view.ID != "" {
			writeJSON(c, status, gin.H{"error": err.Error(), "run": liveRunJSON(view)})
			return
		}
		writeErr(c, status, err)
		return
	}
	writeJSON(c, http.StatusOK, liveRunJSON(view))
}

func (s *Service) consolePlanningCheckStatus(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	eng, err := s.planningLiveEngine()
	if err != nil {
		writeErr(c, http.StatusServiceUnavailable, err)
		return
	}
	view, err := eng.CheckStatus(c.Request.Context(), currentUserID(c), body.ID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrNotFound) {
			status = http.StatusNotFound
		}
		if view.ID != "" {
			writeJSON(c, status, gin.H{"error": err.Error(), "run": liveRunJSON(view)})
			return
		}
		writeErr(c, status, err)
		return
	}
	writeJSON(c, http.StatusOK, liveRunJSON(view))
}

func (s *Service) consolePlanningStop(c *gin.Context) {
	var body struct {
		ID   string `json:"id"`
		Hash string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	eng, err := s.planningLiveEngine()
	if err != nil {
		writeErr(c, http.StatusServiceUnavailable, err)
		return
	}
	view, err := eng.Stop(c.Request.Context(), currentUserID(c), body.ID, body.Hash)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrNotFound) {
			status = http.StatusNotFound
		}
		if view.ID != "" {
			writeJSON(c, status, gin.H{"error": err.Error(), "run": liveRunJSON(view)})
			return
		}
		writeErr(c, status, err)
		return
	}
	writeJSON(c, http.StatusOK, liveRunJSON(view))
}

func (s *Service) consolePlanningReconcile(c *gin.Context) {
	var body struct {
		ID   string `json:"id"`
		Hash string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	eng, err := s.planningLiveEngine()
	if err != nil {
		writeErr(c, http.StatusServiceUnavailable, err)
		return
	}
	view, err := eng.Reconcile(c.Request.Context(), currentUserID(c), body.ID, body.Hash)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrNotFound) {
			status = http.StatusNotFound
		}
		if view.ID != "" {
			writeJSON(c, status, gin.H{"error": err.Error(), "run": liveRunJSON(view)})
			return
		}
		writeErr(c, status, err)
		return
	}
	writeJSON(c, http.StatusOK, liveRunJSON(view))
}

var (
	planningCloseMu       sync.Mutex
	planningCloseRequests = map[string]gin.H{}
)

func writePlanningCloseNotSubmitted(c *gin.Context, err error) {
	writeJSON(c, http.StatusBadRequest, gin.H{
		"error":       err.Error(),
		"close_state": "not_submitted",
	})
}

func (s *Service) consolePlanningClosePosition(c *gin.Context) {
	var body struct {
		ExchangeID string `json:"exchange_id"`
		RequestID  string `json:"request_id"`
		Symbol     string `json:"symbol"`
		Side       string `json:"side"`
		Quantity   string `json:"quantity"`
		EntryPrice string `json:"entry_price"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writePlanningCloseNotSubmitted(c, err)
		return
	}
	userID := currentUserID(c)
	exchangeID := strings.TrimSpace(body.ExchangeID)
	requestID := strings.TrimSpace(body.RequestID)
	symbol := strings.TrimSpace(body.Symbol)
	side := strings.ToLower(strings.TrimSpace(body.Side))
	if exchangeID == "" || requestID == "" || symbol == "" {
		writePlanningCloseNotSubmitted(c, errors.New("exchange_id, request_id and symbol are required"))
		return
	}
	if side != "long" && side != "short" {
		writePlanningCloseNotSubmitted(c, errors.New("side must be long or short"))
		return
	}
	qty, err := strconv.ParseFloat(strings.TrimSpace(body.Quantity), 64)
	if err != nil || qty <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		writePlanningCloseNotSubmitted(c, errors.New("invalid close quantity"))
		return
	}
	entry, err := strconv.ParseFloat(strings.TrimSpace(body.EntryPrice), 64)
	if err != nil || entry <= 0 || math.IsNaN(entry) || math.IsInf(entry, 0) {
		writePlanningCloseNotSubmitted(c, errors.New("invalid entry price"))
		return
	}

	idemKey := userID + "|" + requestID
	planningCloseMu.Lock()
	if cached, ok := planningCloseRequests[idemKey]; ok {
		planningCloseMu.Unlock()
		writeJSON(c, http.StatusOK, cached)
		return
	}
	planningCloseMu.Unlock()

	st := s.eng.Store()
	if st == nil {
		writePlanningCloseNotSubmitted(c, errStoreUnavailable)
		return
	}
	ex, err := st.Exchange().GetByID(userID, exchangeID)
	if err != nil || ex == nil {
		writePlanningCloseNotSubmitted(c, errors.New("exchange account not found"))
		return
	}
	if err := s.assertNoRunningAutopilot(userID, exchangeID); err != nil {
		writePlanningCloseNotSubmitted(c, err)
		return
	}

	trader, err := buildCredentialExchangeTrader(exchangeCredentialFromStore(userID, ex))
	if err != nil {
		writePlanningCloseNotSubmitted(c, err)
		return
	}
	positions, err := trader.GetPositions()
	if err != nil {
		writePlanningCloseNotSubmitted(c, fmt.Errorf("could not read positions before close: %w", err))
		return
	}
	var foundQty, foundEntry float64
	found := false
	for _, pos := range positions {
		posSymbol, _ := pos["symbol"].(string)
		posSide, _ := pos["side"].(string)
		if !live.SamePlanningSymbol(posSymbol, symbol) {
			continue
		}
		if strings.ToLower(posSide) != side {
			continue
		}
		foundQty = absFloat(positionMapFloat(pos, "positionAmt", "quantity", "size"))
		foundEntry = positionMapFloat(pos, "entryPrice", "entry_price")
		found = true
		break
	}
	if !found || foundQty <= 0 {
		writePlanningCloseNotSubmitted(c, errors.New("Position is no longer open; no close was sent."))
		return
	}
	if !live.MatchCloseQuantity(qty, foundQty) || !live.MatchCloseEntry(entry, foundEntry) {
		writePlanningCloseNotSubmitted(c, errors.New("Position changed. Refresh and review its size before closing."))
		return
	}

	var result map[string]any
	switch side {
	case "long":
		result, err = trader.CloseLong(symbol, foundQty)
	default:
		result, err = trader.CloseShort(symbol, foundQty)
	}
	if err != nil {
		// May have reached the venue; do not claim not_submitted.
		writeJSON(c, http.StatusBadGateway, gin.H{
			"error": "Close result is unconfirmed. Check the exchange before retrying.",
		})
		return
	}

	if eng, engErr := s.planningLiveEngine(); engErr == nil {
		_ = eng.MarkSymbolClosed(userID, exchangeID, symbol)
	}

	msg := "Close request accepted. Verify the refreshed position below; no automatic repeat will be sent."
	if result != nil {
		if status, ok := result["status"]; ok {
			msg = fmt.Sprintf("Close accepted (%v). Verify the refreshed position below; no automatic repeat will be sent.", status)
		}
	}
	out := gin.H{"message": msg}
	planningCloseMu.Lock()
	planningCloseRequests[idemKey] = out
	// Bound memory: drop older entries opportunistically.
	if len(planningCloseRequests) > 500 {
		planningCloseRequests = map[string]gin.H{idemKey: out}
	}
	planningCloseMu.Unlock()
	writeJSON(c, http.StatusOK, out)
}

func positionMapFloat(pos map[string]any, keys ...string) float64 {
	for _, key := range keys {
		raw, ok := pos[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case float64:
			return math.Abs(v)
		case float32:
			return math.Abs(float64(v))
		case int:
			return math.Abs(float64(v))
		case int64:
			return math.Abs(float64(v))
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return math.Abs(f)
			}
		}
	}
	return 0
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (s *Service) consolePlanningEntryPlugins(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if c.Request.Method == http.MethodPost {
		var body struct {
			PluginID        string  `json:"plugin_id"`
			Equity          float64 `json:"equity"`
			RiskPct         float64 `json:"risk_pct"`
			Entry           float64 `json:"entry"`
			Stop            float64 `json:"stop"`
			Leverage        float64 `json:"leverage"`
			AvailableMargin float64 `json:"available_margin"`
			MaxNotional     float64 `json:"max_notional"`
			Session         string  `json:"session_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
		id := strings.TrimSpace(body.PluginID)
		if id == "" {
			id = "risk-size-v1"
		}
		result, err := runBuiltinEntryPlugin(id, entryPluginInput{
			Equity:          body.Equity,
			RiskPct:         body.RiskPct,
			Entry:           body.Entry,
			Stop:            body.Stop,
			Leverage:        body.Leverage,
			AvailableMargin: body.AvailableMargin,
			MaxNotional:     body.MaxNotional,
		})
		if err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
		writeJSON(c, http.StatusOK, gin.H{
			"plugin_id":  id,
			"session_id": firstNonEmpty(strings.TrimSpace(body.Session), sessionID),
			"result":     result,
		})
		return
	}
	writeJSON(c, http.StatusOK, builtinEntryPlugins(sessionID))
}

func builtinEntryPlugins(sessionID string) []gin.H {
	return []gin.H{
		{
			"id":          "risk-size-v1",
			"name":        "Risk-based position size",
			"description": "Sizes quantity from equity risk percent, entry/stop distance, and leverage cap.",
			"version":     "1",
			"kind":        "builtin",
			"session_id":  sessionID,
			"inputs":      []string{"equity", "risk_pct", "entry", "stop", "leverage"},
			"outputs":     []string{"quantity", "notional", "margin", "risk_usd", "stop_distance"},
		},
		{
			"id":          "notional-cap-v1",
			"name":        "Notional / margin cap",
			"description": "Caps size by available margin × leverage (or max notional), independent of stop distance.",
			"version":     "1",
			"kind":        "builtin",
			"session_id":  sessionID,
			"inputs":      []string{"available_margin", "leverage", "entry", "max_notional"},
			"outputs":     []string{"quantity", "notional", "margin"},
		},
	}
}

type entryPluginInput struct {
	Equity          float64
	RiskPct         float64
	Entry           float64
	Stop            float64
	Leverage        float64
	AvailableMargin float64
	MaxNotional     float64
}

func runBuiltinEntryPlugin(id string, in entryPluginInput) (gin.H, error) {
	switch id {
	case "risk-size-v1":
		if in.Equity <= 0 {
			return nil, errors.New("equity must be positive")
		}
		riskPct := in.RiskPct
		if riskPct <= 0 {
			riskPct = 1
		}
		if in.Entry <= 0 || in.Stop <= 0 {
			return nil, errors.New("entry and stop must be positive")
		}
		leverage := in.Leverage
		if leverage <= 0 {
			leverage = 1
		}
		stopDist := math.Abs(in.Entry - in.Stop)
		if stopDist <= 0 {
			return nil, errors.New("entry and stop must differ")
		}
		riskUSD := in.Equity * (riskPct / 100)
		qty := riskUSD / stopDist
		notional := qty * in.Entry
		cappedBy := "risk"
		maxNotional := in.Equity * leverage
		if maxNotional > 0 && notional > maxNotional {
			qty = maxNotional / in.Entry
			notional = qty * in.Entry
			riskUSD = qty * stopDist
			cappedBy = "leverage"
		}
		margin := notional / leverage
		return gin.H{
			"quantity":      formatPlanningDecimal(qty),
			"notional":      formatPlanningDecimal(notional),
			"margin":        formatPlanningDecimal(margin),
			"risk_usd":      formatPlanningDecimal(riskUSD),
			"stop_distance": formatPlanningDecimal(stopDist),
			"capped_by":     cappedBy,
		}, nil
	case "notional-cap-v1":
		if in.Entry <= 0 {
			return nil, errors.New("entry must be positive")
		}
		leverage := in.Leverage
		if leverage <= 0 {
			leverage = 1
		}
		available := in.AvailableMargin
		if available <= 0 {
			available = in.Equity
		}
		maxNotional := available * leverage
		if in.MaxNotional > 0 && (maxNotional <= 0 || in.MaxNotional < maxNotional) {
			maxNotional = in.MaxNotional
		}
		if maxNotional <= 0 {
			return nil, errors.New("available margin × leverage must be positive")
		}
		qty := maxNotional / in.Entry
		margin := maxNotional / leverage
		return gin.H{
			"quantity": formatPlanningDecimal(qty),
			"notional": formatPlanningDecimal(maxNotional),
			"margin":   formatPlanningDecimal(margin),
		}, nil
	default:
		return nil, fmt.Errorf("unknown entry plugin %q", id)
	}
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
	instCtx, instCancel := planningInstrumentHTTPTimeout(ctx)
	defer instCancel()
	rules := s.resolvePlanningInstrument(instCtx, venue, symbol, price)
	base, quote := splitSymbolAssets(venue, symbol)
	out := gin.H{
		"venue":        venue,
		"symbol":       symbol,
		"kind":         "perpetual",
		"base_asset":   base,
		"quote_asset":  quote,
		"tick":         rules.Tick,
		"step":         rules.Step,
		"min_qty":      formatPlanningDecimal(rules.MinQty),
		"min_notional": formatPlanningDecimal(rules.MinNotional),
	}
	if rules.ContractValue != "" {
		out["contract_value"] = rules.ContractValue
	}
	writeJSON(c, http.StatusOK, out)
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
