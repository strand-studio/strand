package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"strand/internal/planning/core"
	"strand/internal/provider/hyperliquid"
)

type planningInstrumentRules struct {
	Tick          core.Decimal
	Step          core.Decimal
	MinQty        float64
	MinNotional   float64
	ContractValue core.Decimal // OKX ctVal (base per contract); empty for coin-margined qty venues
}

func (s *Service) resolvePlanningInstrument(ctx context.Context, venue, symbol string, price float64) planningInstrumentRules {
	fallbackTick, fallbackStep, fallbackMinQty, fallbackMinNotional := inferInstrumentRules(price)
	out := planningInstrumentRules{
		Tick:        fallbackTick,
		Step:        fallbackStep,
		MinQty:      fallbackMinQty,
		MinNotional: fallbackMinNotional,
	}
	var resolved *planningInstrumentRules
	var err error
	switch venue {
	case "binance":
		resolved, err = fetchBinanceInstrumentRules(ctx, symbol)
	case "okx":
		resolved, err = fetchOKXInstrumentRules(ctx, symbol)
	case "bitget":
		resolved, err = fetchBitgetInstrumentRules(ctx, symbol)
	case "hyperliquid":
		resolved, err = fetchHyperliquidInstrumentRules(ctx, symbol, price)
	}
	if err == nil && resolved != nil {
		if resolved.Tick != "" {
			out.Tick = resolved.Tick
		}
		if resolved.Step != "" {
			out.Step = resolved.Step
		}
		if resolved.MinQty > 0 {
			out.MinQty = resolved.MinQty
		}
		if resolved.MinNotional > 0 {
			out.MinNotional = resolved.MinNotional
		}
		out.ContractValue = resolved.ContractValue
	}
	return out
}

func fetchBinanceInstrumentRules(ctx context.Context, symbol string) (*planningInstrumentRules, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://fapi.binance.com/fapi/v1/exchangeInfo?symbol="+url.QueryEscape(symbol), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("binance exchangeInfo HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Symbols []struct {
			Symbol  string `json:"symbol"`
			Filters []struct {
				FilterType  string `json:"filterType"`
				TickSize    string `json:"tickSize"`
				StepSize    string `json:"stepSize"`
				MinQty      string `json:"minQty"`
				Notional    string `json:"notional"`
				MinNotional string `json:"minNotional"`
			} `json:"filters"`
		} `json:"symbols"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	for _, sym := range payload.Symbols {
		if !strings.EqualFold(sym.Symbol, symbol) {
			continue
		}
		out := &planningInstrumentRules{MinNotional: 5}
		for _, f := range sym.Filters {
			switch f.FilterType {
			case "PRICE_FILTER":
				if f.TickSize != "" {
					out.Tick = core.Decimal(trimDecimalZeros(f.TickSize))
				}
			case "LOT_SIZE":
				if f.StepSize != "" {
					out.Step = core.Decimal(trimDecimalZeros(f.StepSize))
				}
				if v, e := strconv.ParseFloat(f.MinQty, 64); e == nil && v > 0 {
					out.MinQty = v
				}
			case "MIN_NOTIONAL", "NOTIONAL":
				raw := f.MinNotional
				if raw == "" {
					raw = f.Notional
				}
				if v, e := strconv.ParseFloat(raw, 64); e == nil && v > 0 {
					out.MinNotional = v
				}
			}
		}
		if out.Tick == "" || out.Step == "" {
			return nil, fmt.Errorf("binance filters incomplete for %s", symbol)
		}
		return out, nil
	}
	return nil, fmt.Errorf("binance symbol %s not found", symbol)
}

func fetchOKXInstrumentRules(ctx context.Context, symbol string) (*planningInstrumentRules, error) {
	symbol = strings.TrimSpace(symbol)
	if !strings.Contains(symbol, "-") {
		symbol = strings.ToUpper(strings.TrimSuffix(symbol, "USDT")) + "-USDT-SWAP"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.okx.com/api/v5/public/instruments?instType=SWAP&instId="+url.QueryEscape(symbol), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Code string `json:"code"`
		Data []struct {
			InstId string `json:"instId"`
			TickSz string `json:"tickSz"`
			LotSz  string `json:"lotSz"`
			MinSz  string `json:"minSz"`
			CtVal  string `json:"ctVal"`
			State  string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != "0" || len(payload.Data) == 0 {
		return nil, fmt.Errorf("okx instrument unavailable for %s", symbol)
	}
	row := payload.Data[0]
	if row.State != "" && row.State != "live" {
		return nil, fmt.Errorf("okx instrument %s state=%s", symbol, row.State)
	}
	minQty := 0.0
	if v, e := strconv.ParseFloat(row.MinSz, 64); e == nil {
		minQty = v
	}
	ctVal, _ := strconv.ParseFloat(row.CtVal, 64)
	// Compile works in base asset qty; step/min mirror contract size in base units.
	step := row.LotSz
	if ctVal > 0 {
		if lot, e := strconv.ParseFloat(row.LotSz, 64); e == nil {
			step = strconv.FormatFloat(lot*ctVal, 'f', -1, 64)
		}
		if minQty > 0 {
			minQty = minQty * ctVal
		}
	}
	return &planningInstrumentRules{
		Tick:          core.Decimal(trimDecimalZeros(row.TickSz)),
		Step:          core.Decimal(trimDecimalZeros(step)),
		MinQty:        minQty,
		MinNotional:   5,
		ContractValue: core.Decimal(trimDecimalZeros(row.CtVal)),
	}, nil
}

func fetchBitgetInstrumentRules(ctx context.Context, symbol string) (*planningInstrumentRules, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.bitget.com/api/v2/mix/market/contracts?productType=USDT-FUTURES&symbol="+url.QueryEscape(symbol), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Code string `json:"code"`
		Data []struct {
			Symbol         string `json:"symbol"`
			PricePlace     string `json:"pricePlace"`
			PriceEndStep   string `json:"priceEndStep"`
			SizeMultiplier string `json:"sizeMultiplier"`
			MinTradeNum    string `json:"minTradeNum"`
			MinTradeUSDT   string `json:"minTradeUSDT"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != "00000" || len(payload.Data) == 0 {
		return nil, fmt.Errorf("bitget contract unavailable for %s", symbol)
	}
	row := payload.Data[0]
	places, _ := strconv.Atoi(row.PricePlace)
	tick := row.PriceEndStep
	if places > 0 {
		// Matches bitget/execution.go: tick = priceEndStep * 10^(-pricePlace)
		tick = strconv.FormatFloat(mustParseFloat(row.PriceEndStep)*math.Pow10(-places), 'f', -1, 64)
	}
	minQty := mustParseFloat(row.MinTradeNum)
	minNotional := mustParseFloat(row.MinTradeUSDT)
	if minNotional <= 0 {
		minNotional = 5
	}
	return &planningInstrumentRules{
		Tick:        core.Decimal(trimDecimalZeros(tick)),
		Step:        core.Decimal(trimDecimalZeros(row.SizeMultiplier)),
		MinQty:      minQty,
		MinNotional: minNotional,
	}, nil
}

func fetchHyperliquidInstrumentRules(ctx context.Context, symbol string, price float64) (*planningInstrumentRules, error) {
	coins, err := hyperliquid.GetProvider().GetAllCoins(ctx)
	if err != nil {
		return nil, err
	}
	for _, coin := range coins {
		if !strings.EqualFold(coin.Symbol, symbol) {
			continue
		}
		step := math.Pow10(-coin.SzDecimals)
		if step <= 0 {
			step = 1
		}
		tick, _, minQty, minNotional := inferInstrumentRules(price)
		if minQty < step {
			minQty = step
		}
		return &planningInstrumentRules{
			Tick:        tick,
			Step:        core.Decimal(strconv.FormatFloat(step, 'f', -1, 64)),
			MinQty:      minQty,
			MinNotional: minNotional,
		}, nil
	}
	return nil, fmt.Errorf("hyperliquid symbol %s not found", symbol)
}

func trimDecimalZeros(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func mustParseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func planningInstrumentHTTPTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 8*time.Second)
}
