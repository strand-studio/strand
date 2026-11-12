package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"strand/internal/market"
	"strand/internal/mcp"
	"strand/internal/provider/hyperliquid"
	"strand/internal/provider/vergex"
)

const chatSystemPrompt = `You are Strand Market Copilot — a concise trading research assistant.

You help users with:
- latest market prices and multi-timeframe context
- Vergex / Claw402 direction signals, flow, and liquidation heat
- multi-chain / multi-venue market coverage (Hyperliquid, CEX, Lighter, Aster, Base fees)
- high-level market suggestions (not personalized financial advice)

Rules:
1. Prefer calling tools for live facts. Do not invent prices, rankings, or news.
2. When tools return empty data, say what is missing (e.g. Claw402 wallet not configured).
3. Keep answers structured: short summary, key numbers, then optional caveats.
4. Never place orders or claim you executed a trade.
5. Reply in the same language the user uses.`

// chatToolDefs returns OpenAI-compatible tool schemas for the market copilot.
func chatToolDefs() []mcp.Tool {
	return []mcp.Tool{
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "list_supported_chains",
				Description: "List supported chains, venues, settlement assets, and sample coins.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "list_markets",
				Description: "List tradable symbols for an exchange/venue, optionally filtered by category (crypto/stock/forex/commodity/index).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"exchange": map[string]any{
							"type":        "string",
							"description": "Venue id: hyperliquid, binance, bybit, okx, gate, bitget, lighter, aster, kucoin",
						},
						"category": map[string]any{"type": "string"},
						"limit":    map[string]any{"type": "integer"},
					},
					"required": []string{"exchange"},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "get_klines",
				Description: "Fetch recent OHLCV candles for a symbol.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"symbol":   map[string]any{"type": "string"},
						"exchange": map[string]any{"type": "string"},
						"interval": map[string]any{"type": "string", "description": "e.g. 15m, 1h, 4h, 1d"},
						"limit":    map[string]any{"type": "integer"},
					},
					"required": []string{"symbol"},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "get_direction_leaderboard",
				Description: "Vergex direction-change leaderboard (requires Claw402 wallet).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"limit": map[string]any{"type": "integer"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "get_direction_current",
				Description: "Current Vergex direction bias for one symbol.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"symbol": map[string]any{"type": "string"},
					},
					"required": []string{"symbol"},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "get_flow_markets",
				Description: "Net flow markets ranking from Vergex.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"chain":  map[string]any{"type": "string"},
						"window": map[string]any{"type": "string"},
						"limit":  map[string]any{"type": "integer"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "get_liquidation_heatmap",
				Description: "Cost / liquidation heatmap for a symbol.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"symbol":     map[string]any{"type": "string"},
						"marketType": map[string]any{"type": "string"},
						"chain":      map[string]any{"type": "string"},
						"liqBand":    map[string]any{"type": "string"},
					},
					"required": []string{"symbol"},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "get_account_snapshot",
				Description: "Read-only snapshot of the user's selected trader account equity and positions.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"trader_id": map[string]any{"type": "string"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: mcp.FunctionDef{
				Name:        "market_brief",
				Description: "Compose a compact multi-source brief for one symbol using available local/live data.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"symbol":   map[string]any{"type": "string"},
						"exchange": map[string]any{"type": "string"},
					},
					"required": []string{"symbol"},
				},
			},
		},
	}
}

func argString(args map[string]any, key, fallback string) string {
	if v, ok := args[key]; ok {
		switch t := v.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				return strings.TrimSpace(t)
			}
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64)
		}
	}
	return fallback
}

func argInt(args map[string]any, key string, fallback int) int {
	if v, ok := args[key]; ok {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		case string:
			if n, err := strconv.Atoi(t); err == nil {
				return n
			}
		}
	}
	return fallback
}

func toolResultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"marshal_failed"}`
	}
	return string(b)
}

func (s *Service) executeChatTool(ctx context.Context, userID, name, rawArgs string) string {
	args := decodeJSONObject(rawArgs)
	switch name {
	case "list_supported_chains":
		return toolResultJSON(map[string]any{
			"venues": []string{"hyperliquid", "binance", "bybit", "okx", "gate", "bitget", "lighter", "aster", "kucoin"},
			"chains": []string{"hyperliquid", "base", "ethereum"},
			"notes":  "Base USDC powers Claw402 AI fees; Hyperliquid/HIP-3 covers crypto+stocks+FX+commodities.",
			"sample_coins": map[string]any{
				"crypto": []string{"BTC", "ETH", "SOL"},
				"hip3":   []string{"xyz:TSLA", "xyz:GOLD", "xyz:EUR"},
			},
		})
	case "list_markets":
		exchange := normalizeExchangeQuery(argString(args, "exchange", "hyperliquid"))
		limit := argInt(args, "limit", 40)
		category := strings.ToLower(argString(args, "category", ""))
		symbols, err := s.fetchMarketSymbols(ctx, exchange, category, limit)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		return toolResultJSON(map[string]any{"exchange": exchange, "count": len(symbols), "symbols": symbols})
	case "get_klines":
		symbol := argString(args, "symbol", "")
		if symbol == "" {
			return toolResultJSON(map[string]any{"error": "symbol required"})
		}
		exchange := normalizeExchangeQuery(argString(args, "exchange", "hyperliquid"))
		interval := argString(args, "interval", "1h")
		limit := argInt(args, "limit", 48)
		klines, err := s.fetchKlines(ctx, exchange, symbol, interval, limit)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		return toolResultJSON(map[string]any{
			"symbol": symbol, "exchange": exchange, "interval": interval, "candles": klines,
		})
	case "get_direction_leaderboard":
		client, err := s.resolveVergexClient(userID)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		if client == nil {
			return toolResultJSON(map[string]any{"items": []any{}, "message": "claw402 wallet not configured"})
		}
		data, err := client.GetDirectionChangeLeaderboard(ctx)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		limit := argInt(args, "limit", 15)
		items := data.Items
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		return toolResultJSON(map[string]any{"asOf": data.AsOf, "items": items})
	case "get_direction_current":
		symbol := argString(args, "symbol", "")
		client, err := s.resolveVergexClient(userID)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		if client == nil {
			return toolResultJSON(map[string]any{"symbol": symbol, "message": "claw402 wallet not configured"})
		}
		raw, err := client.GetDirectionChangeCurrent(ctx, symbol)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		return string(raw)
	case "get_flow_markets":
		client, err := s.resolveVergexClient(userID)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		if client == nil {
			return toolResultJSON(map[string]any{"message": "claw402 wallet not configured"})
		}
		raw, err := client.GetFlowMarkets(ctx,
			argString(args, "chain", vergex.DefaultChain),
			argString(args, "window", "1h"),
			argInt(args, "limit", 20),
		)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		return string(raw)
	case "get_liquidation_heatmap":
		client, err := s.resolveVergexClient(userID)
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		if client == nil {
			return toolResultJSON(map[string]any{"message": "claw402 wallet not configured"})
		}
		raw, err := client.GetCostLiquidationHeatmap(ctx, vergex.Query{
			MarketType: argString(args, "marketType", vergex.DefaultMarketType),
			Symbol:     argString(args, "symbol", ""),
			Chain:      argString(args, "chain", vergex.DefaultChain),
			LiqBand:    argString(args, "liqBand", "15"),
		})
		if err != nil {
			return toolResultJSON(map[string]any{"error": err.Error()})
		}
		return string(raw)
	case "get_account_snapshot":
		return s.chatAccountSnapshot(userID, argString(args, "trader_id", ""))
	case "market_brief":
		return s.chatMarketBrief(ctx, userID, argString(args, "symbol", ""), normalizeExchangeQuery(argString(args, "exchange", "hyperliquid")))
	default:
		return toolResultJSON(map[string]any{"error": fmt.Sprintf("unknown tool: %s", name)})
	}
}

func (s *Service) fetchMarketSymbols(ctx context.Context, exchange, category string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 40
	}
	out := make([]map[string]any, 0, limit)
	switch exchange {
	case "hyperliquid":
		coins, err := hyperliquid.GetProvider().GetMainCoins(ctx, limit)
		if err != nil {
			return nil, err
		}
		for _, coin := range coins {
			cat := coinCategory(coin.Symbol)
			if category != "" && cat != category {
				continue
			}
			out = append(out, map[string]any{
				"symbol": coin.Symbol, "category": cat, "mark_price": coin.MarkPrice,
				"change_24h_pct": coin.Change24hPct, "volume_24h": coin.Volume24h,
			})
		}
	default:
		if market.SupportsLiveCEXBoard(exchange) {
			live, err := market.FetchCEXBoard(ctx, exchange, limit)
			if err == nil {
				for _, coin := range live {
					if category != "" && coin.Category != category {
						continue
					}
					out = append(out, map[string]any{
						"symbol": coin.Symbol, "display": coin.Display, "category": coin.Category,
						"mark_price": coin.MarkPrice, "change_24h_pct": coin.Change24hPct,
						"volume_24h": coin.Volume24h, "exchange": exchange,
					})
					if len(out) >= limit {
						break
					}
				}
				if len(out) > 0 {
					return out, nil
				}
			} else {
				market.LogCEXBoardError(exchange, err)
			}
		}
		for _, item := range curatedCEXSymbols {
			cat, _ := item["category"].(string)
			if category != "" && cat != category {
				continue
			}
			row := map[string]any{}
			for k, v := range item {
				row[k] = v
			}
			row["exchange"] = exchange
			out = append(out, row)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *Service) fetchKlines(ctx context.Context, exchange, symbol, interval string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 48
	}
	if limit > 200 {
		limit = 200
	}
	client := hyperliquid.NewClient()
	hlInterval := hyperliquid.MapTimeframe(interval)
	candles, err := client.GetCandles(ctx, symbol, hlInterval, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(candles))
	for _, candle := range candles {
		open, _ := strconv.ParseFloat(candle.Open, 64)
		high, _ := strconv.ParseFloat(candle.High, 64)
		low, _ := strconv.ParseFloat(candle.Low, 64)
		closePx, _ := strconv.ParseFloat(candle.Close, 64)
		vol, _ := strconv.ParseFloat(candle.Volume, 64)
		out = append(out, map[string]any{
			"openTime": candle.OpenTime, "open": open, "high": high, "low": low,
			"close": closePx, "volume": vol, "closeTime": candle.CloseTime,
		})
	}
	_ = exchange
	return out, nil
}

func (s *Service) chatAccountSnapshot(userID, traderID string) string {
	st := s.eng.Store()
	if st == nil {
		return toolResultJSON(map[string]any{"error": "store unavailable"})
	}
	traders, err := st.Trader().List(userID)
	if err != nil {
		return toolResultJSON(map[string]any{"error": err.Error()})
	}
	if len(traders) == 0 {
		return toolResultJSON(map[string]any{"message": "no traders configured"})
	}
	target := traders[0]
	if traderID != "" {
		found := false
		for _, tr := range traders {
			if tr.ID == traderID {
				target = tr
				found = true
				break
			}
		}
		if !found {
			return toolResultJSON(map[string]any{"error": "trader not found"})
		}
	}
	tm := s.eng.Traders()
	if tm == nil {
		return toolResultJSON(map[string]any{
			"trader_id": target.ID, "name": target.Name, "running": target.IsRunning,
			"message": "trader manager unavailable",
		})
	}
	at, err := tm.GetTrader(target.ID)
	if err != nil || at == nil {
		return toolResultJSON(map[string]any{
			"trader_id": target.ID, "name": target.Name, "running": target.IsRunning,
			"message": "trader process not loaded",
		})
	}
	payload := map[string]any{
		"trader_id": target.ID,
		"name":      target.Name,
		"running":   target.IsRunning,
	}
	if account, accErr := at.GetAccountInfo(); accErr == nil {
		payload["account"] = account
	} else {
		payload["account_error"] = accErr.Error()
	}
	if positions, posErr := at.GetPositions(); posErr == nil {
		payload["positions"] = positions
	} else {
		payload["positions_error"] = posErr.Error()
	}
	return toolResultJSON(payload)
}

func (s *Service) chatMarketBrief(ctx context.Context, userID, symbol, exchange string) string {
	if symbol == "" {
		return toolResultJSON(map[string]any{"error": "symbol required"})
	}
	brief := map[string]any{
		"symbol":    symbol,
		"exchange":  exchange,
		"as_of":     time.Now().UTC().Format(time.RFC3339),
		"disclaimer": "Informational only — not investment advice.",
	}
	if klines, err := s.fetchKlines(ctx, exchange, symbol, "1h", 24); err == nil && len(klines) > 0 {
		last := klines[len(klines)-1]
		first := klines[0]
		brief["last_close"] = last["close"]
		brief["open_24h_ago"] = first["open"]
		brief["candles_1h"] = len(klines)
	} else if err != nil {
		brief["klines_error"] = err.Error()
	}
	if client, err := s.resolveVergexClient(userID); err == nil && client != nil {
		if raw, err := client.GetDirectionChangeCurrent(ctx, symbol); err == nil {
			var parsed any
			if json.Unmarshal(raw, &parsed) == nil {
				brief["direction"] = parsed
			} else {
				brief["direction_raw"] = string(raw)
			}
		}
	} else {
		brief["direction"] = "unavailable (configure Claw402)"
	}
	return toolResultJSON(brief)
}
