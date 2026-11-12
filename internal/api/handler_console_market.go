package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"strand/internal/logger"
	"strand/internal/market"
	"strand/internal/provider/hyperliquid"
	"strand/internal/provider/vergex"

	"github.com/gin-gonic/gin"
)

// curatedCEXSymbols is a fallback universe for CEX venues when live exchange-info
// is unavailable. Categories help the Chat / Terminal UIs group markets.
var curatedCEXSymbols = []gin.H{
	{"symbol": "BTCUSDT", "name": "Bitcoin", "category": "crypto", "display": "BTC"},
	{"symbol": "ETHUSDT", "name": "Ethereum", "category": "crypto", "display": "ETH"},
	{"symbol": "SOLUSDT", "name": "Solana", "category": "crypto", "display": "SOL"},
	{"symbol": "BNBUSDT", "name": "BNB", "category": "crypto", "display": "BNB"},
	{"symbol": "XRPUSDT", "name": "XRP", "category": "crypto", "display": "XRP"},
	{"symbol": "DOGEUSDT", "name": "Dogecoin", "category": "crypto", "display": "DOGE"},
	{"symbol": "ADAUSDT", "name": "Cardano", "category": "crypto", "display": "ADA"},
	{"symbol": "AVAXUSDT", "name": "Avalanche", "category": "crypto", "display": "AVAX"},
	{"symbol": "LINKUSDT", "name": "Chainlink", "category": "crypto", "display": "LINK"},
	{"symbol": "DOTUSDT", "name": "Polkadot", "category": "crypto", "display": "DOT"},
	{"symbol": "MATICUSDT", "name": "Polygon", "category": "crypto", "display": "MATIC"},
	{"symbol": "ARBUSDT", "name": "Arbitrum", "category": "crypto", "display": "ARB"},
	{"symbol": "OPUSDT", "name": "Optimism", "category": "crypto", "display": "OP"},
	{"symbol": "SUIUSDT", "name": "Sui", "category": "crypto", "display": "SUI"},
	{"symbol": "APTUSDT", "name": "Aptos", "category": "crypto", "display": "APT"},
	{"symbol": "NEARUSDT", "name": "NEAR", "category": "crypto", "display": "NEAR"},
	{"symbol": "ATOMUSDT", "name": "Cosmos", "category": "crypto", "display": "ATOM"},
	{"symbol": "LTCUSDT", "name": "Litecoin", "category": "crypto", "display": "LTC"},
	{"symbol": "UNIUSDT", "name": "Uniswap", "category": "crypto", "display": "UNI"},
	{"symbol": "AAVEUSDT", "name": "Aave", "category": "crypto", "display": "AAVE"},
}

func normalizeExchangeQuery(raw string) string {
	ex := strings.ToLower(strings.TrimSpace(raw))
	switch ex {
	case "", "hyperliquid-xyz", "hl", "hyperliquid_xyz":
		return "hyperliquid"
	case "okx-swap", "okex":
		return "okx"
	default:
		return ex
	}
}

func coinCategory(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if strings.HasPrefix(s, "XYZ:") || strings.Contains(s, ":") {
		base := s
		if i := strings.Index(base, ":"); i >= 0 {
			base = base[i+1:]
		}
		return hyperliquid.XYZCategory(base)
	}
	return "crypto"
}

func (s *Service) consoleChains(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{
		"chains": []gin.H{
			{
				"id":          "hyperliquid",
				"name":        "Hyperliquid",
				"kind":        "perp_dex",
				"networks":    []string{"mainnet", "testnet"},
				"assets":      []string{"USDC"},
				"categories":  []string{"crypto", "stock", "forex", "commodity", "index", "pre_ipo"},
				"description": "Native perps plus HIP-3 / XYZ multi-asset markets",
			},
			{
				"id":          "base",
				"name":        "Base",
				"kind":        "evm_l2",
				"networks":    []string{"mainnet"},
				"assets":      []string{"USDC", "ETH"},
				"categories":  []string{"payments"},
				"description": "Claw402 AI fee wallet settlement (USDC)",
			},
			{
				"id":          "ethereum",
				"name":        "Ethereum",
				"kind":        "evm_l1",
				"networks":    []string{"mainnet"},
				"assets":      []string{"ETH", "USDC"},
				"categories":  []string{"wallet"},
				"description": "Lighter / Aster wallet-based DEX authentication",
			},
			{
				"id":          "binance",
				"name":        "Binance Futures",
				"kind":        "cex",
				"networks":    []string{"mainnet", "testnet"},
				"assets":      []string{"USDT", "USDC"},
				"categories":  []string{"crypto"},
				"description": "USDT-M perpetual futures via API key",
			},
			{
				"id":          "bybit",
				"name":        "Bybit",
				"kind":        "cex",
				"networks":    []string{"mainnet", "testnet"},
				"assets":      []string{"USDT"},
				"categories":  []string{"crypto"},
				"description": "Linear perpetual futures",
			},
			{
				"id":          "okx",
				"name":        "OKX",
				"kind":        "cex",
				"networks":    []string{"mainnet", "testnet"},
				"assets":      []string{"USDT"},
				"categories":  []string{"crypto"},
				"description": "USDT-margined swaps",
			},
			{
				"id":          "gate",
				"name":        "Gate.io",
				"kind":        "cex",
				"networks":    []string{"mainnet"},
				"assets":      []string{"USDT"},
				"categories":  []string{"crypto"},
				"description": "USDT perpetual contracts",
			},
			{
				"id":          "bitget",
				"name":        "Bitget",
				"kind":        "cex",
				"networks":    []string{"mainnet"},
				"assets":      []string{"USDT"},
				"categories":  []string{"crypto"},
				"description": "USDT-M perpetual futures",
			},
			{
				"id":          "lighter",
				"name":        "Lighter",
				"kind":        "perp_dex",
				"networks":    []string{"mainnet", "testnet"},
				"assets":      []string{"USDC"},
				"categories":  []string{"crypto"},
				"description": "zkLighter perps with EVM wallet auth",
			},
			{
				"id":          "aster",
				"name":        "Aster",
				"kind":        "perp_dex",
				"networks":    []string{"mainnet"},
				"assets":      []string{"USDC"},
				"categories":  []string{"crypto"},
				"description": "Aster DEX via API Pro wallet",
			},
			{
				"id":          "kucoin",
				"name":        "KuCoin",
				"kind":        "cex",
				"networks":    []string{"mainnet"},
				"assets":      []string{"USDT"},
				"categories":  []string{"crypto"},
				"description": "KuCoin futures (passphrase auth)",
			},
		},
		"coins": gin.H{
			"crypto_majors": []string{"BTC", "ETH", "SOL", "BNB", "XRP", "DOGE", "ADA", "AVAX", "LINK", "DOT", "SUI", "ARB", "OP", "APT", "NEAR"},
			"hip3_samples":  []string{"xyz:TSLA", "xyz:NVDA", "xyz:GOLD", "xyz:EUR", "xyz:SPX"},
		},
	})
}

func (s *Service) consoleSymbols(c *gin.Context) {
	exchange := normalizeExchangeQuery(c.DefaultQuery("exchange", "hyperliquid"))
	limit := 500
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	categoryFilter := strings.ToLower(strings.TrimSpace(c.Query("category")))

	timeout := 20 * time.Second
	if market.SupportsLiveCEXBoard(exchange) {
		timeout = 35 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	symbols := make([]gin.H, 0)
	switch exchange {
	case "hyperliquid":
		coins, err := hyperliquid.GetProvider().GetAllCoins(ctx)
		if err != nil {
			writeErr(c, http.StatusBadGateway, err)
			return
		}
		xyz, xyzErr := hyperliquid.GetPerpDexCoins(ctx, "xyz")
		if xyzErr == nil {
			coins = append(coins, xyz...)
		}
		for _, coin := range coins {
			cat := coinCategory(coin.Symbol)
			if categoryFilter != "" && cat != categoryFilter {
				continue
			}
			symbols = append(symbols, gin.H{
				"symbol":         coin.Symbol,
				"display":        coin.Symbol,
				"name":           coin.Symbol,
				"category":       cat,
				"exchange":       "hyperliquid",
				"volume_24h":     coin.Volume24h,
				"mark_price":     coin.MarkPrice,
				"change_24h_pct": coin.Change24hPct,
				"prev_day_price": coin.PrevDayPrice,
				"maxLeverage":    coin.MaxLeverage,
				"sz_decimals":    coin.SzDecimals,
			})
			if len(symbols) >= limit {
				break
			}
		}
	case "binance", "bybit", "okx", "gate", "bitget", "kucoin":
		source := "live"
		if market.SupportsLiveCEXBoard(exchange) {
			live, liveErr := market.FetchCEXBoard(ctx, exchange, limit*2)
			if liveErr != nil {
				market.LogCEXBoardError(exchange, liveErr)
			} else {
				for _, coin := range live {
					if categoryFilter != "" && coin.Category != categoryFilter {
						continue
					}
					symbols = append(symbols, gin.H{
						"symbol":         coin.Symbol,
						"display":        coin.Display,
						"name":           coin.Name,
						"category":       coin.Category,
						"exchange":       exchange,
						"volume_24h":     coin.Volume24h,
						"mark_price":     coin.MarkPrice,
						"change_24h_pct": coin.Change24hPct,
						"prev_day_price": coin.PrevDayPrice,
					})
					if len(symbols) >= limit {
						break
					}
				}
			}
		}
		if len(symbols) == 0 {
			source = "curated"
			for _, item := range curatedCEXSymbols {
				cat, _ := item["category"].(string)
				if categoryFilter != "" && cat != categoryFilter {
					continue
				}
				row := gin.H{}
				for k, v := range item {
					row[k] = v
				}
				row["exchange"] = exchange
				// OKX uses SWAP style symbols in Frontier / trading clients.
				if exchange == "okx" {
					sym, _ := item["display"].(string)
					if sym == "" {
						sym, _ = item["symbol"].(string)
						sym = strings.TrimSuffix(sym, "USDT")
					}
					row["symbol"] = sym + "-USDT-SWAP"
					row["display"] = sym
				}
				symbols = append(symbols, row)
				if len(symbols) >= limit {
					break
				}
			}
		}
		writeJSON(c, http.StatusOK, gin.H{
			"exchange": exchange,
			"symbols":  symbols,
			"count":    len(symbols),
			"source":   source,
		})
		return
	case "lighter", "aster":
		for _, item := range curatedCEXSymbols {
			cat, _ := item["category"].(string)
			if categoryFilter != "" && cat != categoryFilter {
				continue
			}
			row := gin.H{}
			for k, v := range item {
				row[k] = v
			}
			row["exchange"] = exchange
			symbols = append(symbols, row)
			if len(symbols) >= limit {
				break
			}
		}
		writeJSON(c, http.StatusOK, gin.H{
			"exchange": exchange,
			"symbols":  symbols,
			"count":    len(symbols),
			"source":   "curated",
		})
		return
	default:
		writeJSON(c, http.StatusOK, gin.H{
			"exchange": exchange,
			"symbols":  []any{},
			"count":    0,
			"error":    "unsupported exchange",
		})
		return
	}

	writeJSON(c, http.StatusOK, gin.H{
		"exchange": exchange,
		"symbols":  symbols,
		"count":    len(symbols),
		"source":   "live",
	})
}

func (s *Service) consoleKlines(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}
	interval := c.DefaultQuery("interval", "1h")
	exchange := normalizeExchangeQuery(c.DefaultQuery("exchange", "hyperliquid"))
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	var out []gin.H
	switch exchange {
	case "hyperliquid", "lighter", "aster":
		client := hyperliquid.NewClient()
		hlInterval := hyperliquid.MapTimeframe(interval)
		candles, err := client.GetCandles(ctx, symbol, hlInterval, limit)
		if err != nil {
			writeErr(c, http.StatusBadGateway, err)
			return
		}
		out = make([]gin.H, 0, len(candles))
		for _, candle := range candles {
			open, _ := strconv.ParseFloat(candle.Open, 64)
			high, _ := strconv.ParseFloat(candle.High, 64)
			low, _ := strconv.ParseFloat(candle.Low, 64)
			closePx, _ := strconv.ParseFloat(candle.Close, 64)
			vol, _ := strconv.ParseFloat(candle.Volume, 64)
			out = append(out, gin.H{
				"openTime":  candle.OpenTime,
				"open":      open,
				"high":      high,
				"low":       low,
				"close":     closePx,
				"volume":    vol,
				"closeTime": candle.CloseTime,
			})
		}
	case "binance", "bybit", "okx", "gate", "bitget", "kucoin":
		candles, err := market.FetchCEXKlines(ctx, exchange, symbol, interval, limit)
		if err != nil {
			market.LogCEXBoardError(exchange+" klines", err)
			writeJSON(c, http.StatusOK, []any{})
			return
		}
		out = make([]gin.H, 0, len(candles))
		for _, k := range candles {
			out = append(out, gin.H{
				"openTime":  k.OpenTime,
				"open":      k.Open,
				"high":      k.High,
				"low":       k.Low,
				"close":     k.Close,
				"volume":    k.Volume,
				"closeTime": k.CloseTime,
			})
		}
	default:
		writeJSON(c, http.StatusOK, []any{})
		return
	}
	writeJSON(c, http.StatusOK, out)
}

func (s *Service) resolveVergexClient(userID string) (*vergex.Client, error) {
	st := s.eng.Store()
	walletKey := ""
	if st != nil {
		key, err := st.AIModel().ResolveClaw402WalletKey(userID, "")
		if err == nil {
			walletKey = strings.TrimSpace(key)
		}
	}
	if walletKey == "" {
		walletKey = strings.TrimSpace(os.Getenv("CLAW402_WALLET_KEY"))
	}
	if walletKey == "" {
		return nil, nil
	}
	baseURL := strings.TrimSpace(os.Getenv("CLAW402_URL"))
	if baseURL == "" {
		baseURL = vergex.DefaultBaseURL
	}
	return vergex.NewClient(baseURL, walletKey, &logger.MCPLogger{})
}

func (s *Service) consoleVergexLeaderboard(c *gin.Context) {
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{
			"band": 0, "universeSize": 0, "rankBy": "", "asOf": 0, "items": []any{},
			"message": "configure claw402 wallet to unlock Vergex signals",
		})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	data, err := client.GetDirectionChangeLeaderboard(ctx)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	items := make([]gin.H, 0, len(data.Items))
	for _, item := range data.Items {
		items = append(items, gin.H{
			"rank":         item.Rank,
			"symbol":       item.Symbol,
			"bias":         item.Bias,
			"directionScore": item.Score,
			"bullishCount": item.BullishCount,
			"bearishCount": item.BearishCount,
			"neutralCount": item.NeutralCount,
			"markPrice":    item.MarkPrice,
			"oiRank":       item.OIRank,
			"market": gin.H{
				"marketType": item.MarketType,
				"symbol":     item.Symbol,
			},
		})
	}
	writeJSON(c, http.StatusOK, gin.H{
		"band":         data.Band,
		"universeSize": data.UniverseSize,
		"rankBy":       data.RankBy,
		"asOf":         data.AsOf,
		"items":        items,
	})
}

func (s *Service) consoleVergexCurrent(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{"symbol": symbol, "direction": "unknown"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	raw, err := client.GetDirectionChangeCurrent(ctx, symbol)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

func (s *Service) consoleVergexHistory(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	eventType := c.DefaultQuery("eventType", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{"items": []any{}})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	raw, err := client.GetDirectionChangeHistory(ctx, symbol, eventType, page, pageSize)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

func (s *Service) consoleVergexHeatmap(c *gin.Context) {
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	q := vergex.Query{
		MarketType: c.DefaultQuery("marketType", vergex.DefaultMarketType),
		Symbol:     c.Query("symbol"),
		Chain:      c.DefaultQuery("chain", vergex.DefaultChain),
		LiqBand:    c.DefaultQuery("liqBand", "15"),
		Category:   c.Query("category"),
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	raw, err := client.GetCostLiquidationHeatmap(ctx, q)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

func (s *Service) consoleVergexWinrate(c *gin.Context) {
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	q := vergex.WinrateQuery{
		MarketType: c.DefaultQuery("marketType", vergex.DefaultMarketType),
		Symbol:     c.Query("symbol"),
		Chain:      c.DefaultQuery("chain", vergex.DefaultChain),
	}
	if v := c.Query("winMin"); v != "" {
		q.WinMin, _ = strconv.Atoi(v)
	}
	if v := c.Query("winMax"); v != "" {
		q.WinMax, _ = strconv.Atoi(v)
	}
	if v := c.Query("costMin"); v != "" {
		q.CostMin, _ = strconv.ParseFloat(v, 64)
	}
	if v := c.Query("costMax"); v != "" {
		q.CostMax, _ = strconv.ParseFloat(v, 64)
	}
	if v := c.Query("minRoundTrips"); v != "" {
		q.MinRoundTrips, _ = strconv.Atoi(v)
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	raw, err := client.GetHolderWinrateMap(ctx, q)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

func (s *Service) consoleVergexWinrateHolders(c *gin.Context) {
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{})
		return
	}
	q := vergex.WinrateHoldersQuery{
		WinrateQuery: vergex.WinrateQuery{
			MarketType: c.DefaultQuery("marketType", vergex.DefaultMarketType),
			Symbol:     c.Query("symbol"),
			Chain:      c.DefaultQuery("chain", vergex.DefaultChain),
		},
		SnapshotID: c.Query("snapshotId"),
		Side:       c.Query("side"),
	}
	q.Row, _ = strconv.Atoi(c.DefaultQuery("row", "0"))
	q.RowEnd, _ = strconv.Atoi(c.DefaultQuery("rowEnd", "0"))
	q.Column, _ = strconv.Atoi(c.DefaultQuery("column", "0"))
	q.ColumnEnd, _ = strconv.Atoi(c.DefaultQuery("columnEnd", "0"))
	q.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	q.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	raw, err := client.GetHolderWinrateHolders(ctx, q)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

func (s *Service) consoleVergexFlowMarkets(c *gin.Context) {
	client, err := s.resolveVergexClient(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	if client == nil {
		writeJSON(c, http.StatusOK, gin.H{"items": []any{}})
		return
	}
	chain := c.DefaultQuery("chain", vergex.DefaultChain)
	window := c.DefaultQuery("window", "1h")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	raw, err := client.GetFlowMarkets(ctx, chain, window, limit)
	if err != nil {
		writeErr(c, http.StatusBadGateway, err)
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

// decodeJSONObject is a tiny helper for chat tool tests / handlers.
func decodeJSONObject(raw string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
