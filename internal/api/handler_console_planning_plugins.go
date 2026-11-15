package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"strand/internal/market"
	"strand/internal/provider/vergex"

	"github.com/gin-gonic/gin"
)

type planningPluginManifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

var (
	planningPluginMu    sync.Mutex
	planningPluginPrefs = map[string]map[string]bool{}
)

func defaultPlanningPlugins() []planningPluginManifest {
	return []planningPluginManifest{
		{
			ID:          "funding",
			Name:        "Funding rate",
			Description: "Latest perpetual funding rate (public exchange data).",
			Enabled:     true,
		},
		{
			ID:          "open_interest",
			Name:        "Open interest",
			Description: "Open interest snapshot for the selected market (public data).",
			Enabled:     true,
		},
		{
			ID:          "hyperdata_cost",
			Name:        "Cost and liquidation",
			Description: "Claw402 / Vergex holder cost and liquidation heatmap (Hyperliquid).",
			Enabled:     false,
		},
		{
			ID:          "hyperdata_profit",
			Name:        "Direction leaders",
			Description: "Claw402 / Vergex direction-change leaders (proxy ranking for Trade).",
			Enabled:     false,
		},
		{
			ID:          "hyperdata_winrate",
			Name:        "Holder win rate",
			Description: "Claw402 / Vergex holder win-rate map (Hyperliquid).",
			Enabled:     false,
		},
	}
}

func (s *Service) planningPluginsFor(userID string) []planningPluginManifest {
	base := defaultPlanningPlugins()
	planningPluginMu.Lock()
	defer planningPluginMu.Unlock()
	prefs := planningPluginPrefs[userID]
	if prefs == nil {
		return base
	}
	out := make([]planningPluginManifest, len(base))
	for i, item := range base {
		out[i] = item
		if enabled, ok := prefs[item.ID]; ok {
			out[i].Enabled = enabled
		}
	}
	return out
}

func (s *Service) consolePlanningDataPlugins(c *gin.Context) {
	userID := currentUserID(c)
	if c.Request.Method == http.MethodGet {
		writeJSON(c, http.StatusOK, s.planningPluginsFor(userID))
		return
	}
	var body struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	id := strings.TrimSpace(body.ID)
	known := false
	for _, item := range defaultPlanningPlugins() {
		if item.ID == id {
			known = true
			break
		}
	}
	if !known {
		writeErr(c, http.StatusBadRequest, fmt.Errorf("unknown data plugin %q", id))
		return
	}
	planningPluginMu.Lock()
	if planningPluginPrefs[userID] == nil {
		planningPluginPrefs[userID] = map[string]bool{}
	}
	planningPluginPrefs[userID][id] = body.Enabled
	planningPluginMu.Unlock()
	writeJSON(c, http.StatusOK, s.planningPluginsFor(userID))
}

func (s *Service) consolePlanningPluginData(c *gin.Context) {
	userID := currentUserID(c)
	plugin := strings.TrimSpace(c.Query("plugin"))
	venue := normalizeExchangeQuery(c.DefaultQuery("venue", "binance"))
	symbol := strings.TrimSpace(c.Query("symbol"))
	window := strings.TrimSpace(c.DefaultQuery("window", "7d"))
	if plugin == "" || symbol == "" {
		writeErr(c, http.StatusBadRequest, errors.New("plugin and symbol required"))
		return
	}
	enabled := false
	for _, item := range s.planningPluginsFor(userID) {
		if item.ID == plugin {
			enabled = item.Enabled
			break
		}
	}
	if !enabled {
		writeErr(c, http.StatusBadRequest, errors.New("enable this data plugin before reading it"))
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	switch plugin {
	case "funding", "open_interest":
		marketSymbol := planningPublicMarketSymbol(venue, symbol)
		data, err := market.Get(marketSymbol)
		if err != nil {
			writeErr(c, http.StatusBadGateway, err)
			return
		}
		metrics := []gin.H{}
		notice := "Public market data · not a paid Claw402 read."
		if plugin == "funding" {
			metrics = append(metrics, gin.H{
				"key": "funding_rate", "label": "Funding rate",
				"value": strconv.FormatFloat(data.FundingRate, 'f', -1, 64), "unit": "fraction",
			})
		} else {
			oiLatest, oiAvg := 0.0, 0.0
			if data.OpenInterest != nil {
				oiLatest = data.OpenInterest.Latest
				oiAvg = data.OpenInterest.Average
			}
			metrics = append(metrics,
				gin.H{"key": "oi_latest", "label": "Open interest", "value": strconv.FormatFloat(oiLatest, 'f', -1, 64), "unit": "contracts"},
				gin.H{"key": "oi_average", "label": "OI average", "value": strconv.FormatFloat(oiAvg, 'f', -1, 64), "unit": "contracts"},
			)
			if data.OpenInterest == nil {
				notice = "Open interest was unavailable for this symbol; values are zero."
			}
		}
		writeJSON(c, http.StatusOK, gin.H{
			"plugin":   plugin,
			"venue":    venue,
			"symbol":   symbol,
			"source":   "public_market",
			"observed": now,
			"notice":   notice,
			"metrics":  metrics,
		})
	case "hyperdata_cost", "hyperdata_profit", "hyperdata_winrate":
		if venue != "hyperliquid" {
			writeJSON(c, http.StatusOK, gin.H{
				"plugin": plugin, "venue": venue, "symbol": symbol,
				"source": "unavailable", "observed": now,
				"notice":  "Holder analytics require a Hyperliquid market.",
				"metrics": []any{},
			})
			return
		}
		client, err := s.resolveVergexClient(userID)
		if err != nil {
			writeErr(c, http.StatusBadGateway, err)
			return
		}
		if client == nil {
			writeJSON(c, http.StatusOK, gin.H{
				"plugin": plugin, "venue": venue, "symbol": symbol,
				"source": "unavailable", "observed": now,
				"notice":  "Configure a Claw402 fee wallet to unlock Vergex holder analytics.",
				"metrics": []any{},
			})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()
		payload, perr := s.fetchPlanningHyperdata(ctx, client, plugin, symbol, window)
		if perr != nil {
			writeJSON(c, http.StatusOK, gin.H{
				"plugin": plugin, "venue": venue, "symbol": symbol,
				"source": "vergex", "observed": now,
				"notice":  perr.Error(),
				"metrics": []any{},
			})
			return
		}
		payload["plugin"] = plugin
		payload["venue"] = venue
		payload["symbol"] = symbol
		payload["observed"] = now
		writeJSON(c, http.StatusOK, payload)
	default:
		writeErr(c, http.StatusBadRequest, fmt.Errorf("unsupported plugin %q", plugin))
	}
}

func (s *Service) fetchPlanningHyperdata(ctx context.Context, client *vergex.Client, plugin, symbol, window string) (gin.H, error) {
	switch plugin {
	case "hyperdata_cost":
		raw, err := client.GetCostLiquidationHeatmap(ctx, vergex.Query{
			MarketType: vergex.DefaultMarketType,
			Symbol:     symbol,
			Chain:      vergex.DefaultChain,
			LiqBand:    "15",
		})
		if err != nil {
			return nil, err
		}
		return adaptVergexHeatmap(raw)
	case "hyperdata_winrate":
		raw, err := client.GetHolderWinrateMap(ctx, vergex.WinrateQuery{
			MarketType: vergex.DefaultMarketType,
			Symbol:     symbol,
			Chain:      vergex.DefaultChain,
		})
		if err != nil {
			return nil, err
		}
		return adaptVergexWinrate(raw)
	case "hyperdata_profit":
		_ = window
		data, err := client.GetDirectionChangeLeaderboard(ctx)
		if err != nil {
			return nil, err
		}
		return adaptVergexLeaders(data, symbol)
	default:
		return nil, fmt.Errorf("unsupported hyperdata plugin")
	}
}

func planningPublicMarketSymbol(venue, symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	s = strings.TrimPrefix(s, "XYZ:")
	switch venue {
	case "hyperliquid", "lighter", "aster":
		if !strings.HasSuffix(s, "USDT") {
			s += "USDT"
		}
	case "okx":
		s = strings.ReplaceAll(s, "-", "")
		s = strings.TrimSuffix(s, "SWAP")
	default:
		s = strings.ReplaceAll(s, "-", "")
		s = strings.TrimSuffix(s, "SWAP")
	}
	return market.Normalize(s)
}

func decodeVergexMap(raw json.RawMessage) map[string]any {
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil
	}
	if data, ok := top["data"].(map[string]any); ok {
		return data
	}
	return top
}

func adaptVergexHeatmap(raw json.RawMessage) (gin.H, error) {
	data := decodeVergexMap(raw)
	if data == nil {
		return nil, errors.New("invalid heatmap payload")
	}
	binsRaw, _ := data["bins"].([]any)
	bins := make([]gin.H, 0, len(binsRaw))
	var totalLong, totalShort float64
	for _, item := range binsRaw {
		bin, ok := item.(map[string]any)
		if !ok {
			continue
		}
		px := firstMapFloat(bin, "px", "price", "bucketStartPrice")
		longCost := firstMapFloat(bin, "longCost")
		shortCost := firstMapFloat(bin, "shortCost")
		longLiq := firstMapFloat(bin, "longLiq", "longLiquidation")
		shortLiq := firstMapFloat(bin, "shortLiq", "shortLiquidation")
		totalLong += longCost
		totalShort += shortCost
		bins = append(bins, gin.H{
			"px": px, "longCost": longCost, "shortCost": shortCost,
			"longLiq": longLiq, "shortLiq": shortLiq,
		})
	}
	mark := firstMapFloat(data, "currentPrice", "price", "markPrice")
	state := "ready"
	if len(bins) == 0 {
		state = "empty"
	}
	return gin.H{
		"source": "vergex",
		"notice": "Vergex cost / liquidation heatmap via Claw402.",
		"metrics": []gin.H{
			{"key": "long_cost", "label": "Long cost total", "value": strconv.FormatFloat(totalLong, 'f', 2, 64), "unit": "USDC"},
			{"key": "short_cost", "label": "Short cost total", "value": strconv.FormatFloat(totalShort, 'f', 2, 64), "unit": "USDC"},
			{"key": "mark", "label": "Mark", "value": strconv.FormatFloat(mark, 'f', -1, 64), "unit": "USDC"},
		},
		"visual": gin.H{
			"kind": "riskbins", "state": state, "coverage": "heatmap",
			"data_at": time.Now().UTC().Format(time.RFC3339),
			"mark": mark, "bins": bins, "rows": 0, "columns": 0,
			"win_min": 0, "win_max": 0, "cost_min": 0, "cost_max": 0,
		},
	}, nil
}

func adaptVergexWinrate(raw json.RawMessage) (gin.H, error) {
	data := decodeVergexMap(raw)
	if data == nil {
		return nil, errors.New("invalid winrate payload")
	}
	if errObj, ok := data["error"].(map[string]any); ok {
		msg := firstMapString(errObj, "message", "code", "reason")
		if msg == "" {
			msg = "win-rate matrix unavailable"
		}
		return nil, errors.New(msg)
	}
	cellsRaw, _ := data["cells"].([]any)
	cells := make([]gin.H, 0, len(cellsRaw))
	for _, item := range cellsRaw {
		cell, ok := item.(map[string]any)
		if !ok {
			continue
		}
		long := mapAny(cell["long"])
		short := mapAny(cell["short"])
		cells = append(cells, gin.H{
			"row":    firstMapInt(cell, "row"),
			"column": firstMapInt(cell, "column"),
			"long": gin.H{
				"count":    firstMapInt(long, "count"),
				"notional": strconv.FormatFloat(firstMapFloat(long, "notional"), 'f', 2, 64),
			},
			"short": gin.H{
				"count":    firstMapInt(short, "count"),
				"notional": strconv.FormatFloat(firstMapFloat(short, "notional"), 'f', 2, 64),
			},
		})
	}
	viewport := mapAny(data["viewport"])
	winMin := firstMapInt(data, "winMin")
	winMax := firstMapInt(data, "winMax")
	if winMax <= winMin {
		winMin, winMax = 0, 100
	}
	costMin := firstMapFloat(viewport, "costMin")
	costMax := firstMapFloat(viewport, "costMax")
	if costMax <= costMin {
		costMin, costMax = 60, 140
	}
	rows := firstMapInt(data, "winBins")
	cols := firstMapInt(data, "costBins")
	if rows <= 0 {
		rows = 20
	}
	if cols <= 0 {
		cols = 16
	}
	mark := firstMapFloat(data, "markPrice", "currentPrice", "price")
	state := "ready"
	if len(cells) == 0 {
		state = "empty"
	}
	return gin.H{
		"source": "vergex",
		"notice": "Vergex holder win-rate map via Claw402.",
		"metrics": []gin.H{
			{"key": "cells", "label": "Cells", "value": strconv.Itoa(len(cells)), "unit": "count"},
			{"key": "mark", "label": "Mark", "value": strconv.FormatFloat(mark, 'f', -1, 64), "unit": "USDC"},
		},
		"visual": gin.H{
			"kind": "winrate-map", "state": state, "coverage": "matrix",
			"data_at": time.Now().UTC().Format(time.RFC3339),
			"mark": mark, "cells": cells, "rows": rows, "columns": cols,
			"win_min": winMin, "win_max": winMax, "cost_min": costMin, "cost_max": costMax,
		},
	}, nil
}

func adaptVergexLeaders(data *vergex.DirectionChangeLeaderboardData, symbol string) (gin.H, error) {
	if data == nil {
		return nil, errors.New("empty leaderboard")
	}
	entries := make([]gin.H, 0, len(data.Items))
	matchedScore := 0.0
	for i, item := range data.Items {
		rank := item.Rank
		if rank <= 0 {
			rank = i + 1
		}
		entries = append(entries, gin.H{
			"rank":         rank,
			"address":      item.Symbol,
			"netPnl":       strconv.FormatFloat(item.Score, 'f', 4, 64),
			"realizedPnl":  strconv.FormatFloat(item.MarkPrice, 'f', 4, 64),
			"fees":         "0",
			"funding":      item.Bias,
			"volume":       strconv.Itoa(item.BullishCount + item.BearishCount),
		})
		if strings.EqualFold(item.Symbol, symbol) || strings.EqualFold(item.APISymbol, symbol) {
			matchedScore = item.Score
		}
		if len(entries) >= 25 {
			break
		}
	}
	return gin.H{
		"source": "vergex",
		"notice": "Direction-change leaders (not cash PnL). Use as a ranking shortcut for Hyperliquid markets.",
		"metrics": []gin.H{
			{"key": "universe", "label": "Universe", "value": strconv.Itoa(data.UniverseSize), "unit": "count"},
			{"key": "selected_score", "label": "Selected score", "value": strconv.FormatFloat(matchedScore, 'f', 4, 64), "unit": "score"},
		},
		"visual": gin.H{
			"kind": "profit-leaderboard", "state": "ready", "coverage": "leaderboard",
			"data_at": time.Unix(data.AsOf, 0).UTC().Format(time.RFC3339),
			"mark": 0, "rows": 0, "columns": 0,
			"win_min": 0, "win_max": 0, "cost_min": 0, "cost_max": 0,
			"entries": entries,
		},
	}, nil
}

func mapAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func firstMapFloat(m map[string]any, keys ...string) float64 {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case json.Number:
			f, _ := v.Float64()
			return f
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return f
			}
		}
	}
	return 0
}

func firstMapInt(m map[string]any, keys ...string) int {
	return int(firstMapFloat(m, keys...))
}

func firstMapString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
