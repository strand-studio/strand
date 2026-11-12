package market

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Candle is a normalized OHLCV bar used by the Markets detail page.
type Candle struct {
	OpenTime  int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime int64
}

func mapCEXInterval(exchange, interval string) string {
	iv := strings.ToLower(strings.TrimSpace(interval))
	switch exchange {
	case "bybit":
		switch iv {
		case "1m":
			return "1"
		case "3m":
			return "3"
		case "5m":
			return "5"
		case "15m":
			return "15"
		case "30m":
			return "30"
		case "1h":
			return "60"
		case "2h":
			return "120"
		case "4h":
			return "240"
		case "6h":
			return "360"
		case "12h":
			return "720"
		case "1d", "1D":
			return "D"
		case "1w", "1W":
			return "W"
		default:
			return "60"
		}
	case "okx":
		switch iv {
		case "1m":
			return "1m"
		case "3m":
			return "3m"
		case "5m":
			return "5m"
		case "15m":
			return "15m"
		case "30m":
			return "30m"
		case "1h":
			return "1H"
		case "2h":
			return "2H"
		case "4h":
			return "4H"
		case "6h":
			return "6H"
		case "12h":
			return "12H"
		case "1d":
			return "1D"
		case "1w":
			return "1W"
		default:
			return "1H"
		}
	case "gate":
		switch iv {
		case "1m":
			return "1m"
		case "5m":
			return "5m"
		case "15m":
			return "15m"
		case "30m":
			return "30m"
		case "1h":
			return "1h"
		case "4h":
			return "4h"
		case "1d":
			return "1d"
		case "1w":
			return "7d"
		default:
			return "1h"
		}
	case "bitget":
		switch iv {
		case "1m":
			return "1m"
		case "5m":
			return "5m"
		case "15m":
			return "15m"
		case "30m":
			return "30m"
		case "1h":
			return "1H"
		case "4h":
			return "4H"
		case "6h":
			return "6H"
		case "12h":
			return "12H"
		case "1d":
			return "1D"
		case "1w":
			return "1W"
		default:
			return "1H"
		}
	case "kucoin":
		switch iv {
		case "1m":
			return "1"
		case "5m":
			return "5"
		case "15m":
			return "15"
		case "30m":
			return "30"
		case "1h":
			return "60"
		case "4h":
			return "240"
		case "1d":
			return "1440"
		case "1w":
			return "10080"
		default:
			return "60"
		}
	default: // binance-shaped
		switch iv {
		case "1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d", "1w":
			return iv
		default:
			return "1h"
		}
	}
}

// FetchCEXKlines loads public perpetual klines for supported CEX venues.
func FetchCEXKlines(ctx context.Context, exchange, symbol, interval string, limit int) ([]Candle, error) {
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol required")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	mapped := mapCEXInterval(exchange, interval)

	switch exchange {
	case "binance":
		return fetchBinanceKlines(ctx, symbol, mapped, limit)
	case "bybit":
		return fetchBybitKlines(ctx, symbol, mapped, limit)
	case "okx":
		return fetchOKXKlines(ctx, symbol, mapped, limit)
	case "gate":
		return fetchGateKlines(ctx, symbol, mapped, limit)
	case "bitget":
		return fetchBitgetKlines(ctx, symbol, mapped, limit)
	case "kucoin":
		return fetchKucoinKlines(ctx, symbol, mapped, limit)
	default:
		return nil, fmt.Errorf("klines unsupported for exchange %s", exchange)
	}
}

func fetchBinanceKlines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	url := fmt.Sprintf(
		"https://fapi.binance.com/fapi/v1/klines?symbol=%s&interval=%s&limit=%d",
		symbol, interval, limit,
	)
	var raw [][]any
	if err := getBoardJSONFirst(ctx, []string{
		url,
		fmt.Sprintf("https://fapi1.binance.com/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", symbol, interval, limit),
	}, &raw); err != nil {
		return nil, err
	}
	out := make([]Candle, 0, len(raw))
	for _, row := range raw {
		if len(row) < 7 {
			continue
		}
		openTime := anyToInt64(row[0])
		closeTime := anyToInt64(row[6])
		out = append(out, Candle{
			OpenTime:  openTime,
			Open:      anyToFloat(row[1]),
			High:      anyToFloat(row[2]),
			Low:       anyToFloat(row[3]),
			Close:     anyToFloat(row[4]),
			Volume:    anyToFloat(row[5]),
			CloseTime: closeTime,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("binance returned no klines")
	}
	return out, nil
}

func fetchBybitKlines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	url := fmt.Sprintf(
		"https://api.bybit.com/v5/market/kline?category=linear&symbol=%s&interval=%s&limit=%d",
		symbol, interval, limit,
	)
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}
	if err := getBoardJSONFirst(ctx, []string{url, strings.Replace(url, "api.bybit.com", "api.bytick.com", 1)}, &raw); err != nil {
		return nil, err
	}
	if raw.RetCode != 0 {
		return nil, fmt.Errorf("bybit: %s", raw.RetMsg)
	}
	// Bybit returns newest-first.
	out := make([]Candle, 0, len(raw.Result.List))
	for i := len(raw.Result.List) - 1; i >= 0; i-- {
		row := raw.Result.List[i]
		if len(row) < 6 {
			continue
		}
		openTime := parseBoardFloat(row[0])
		out = append(out, Candle{
			OpenTime: int64(openTime),
			Open:     parseBoardFloat(row[1]),
			High:     parseBoardFloat(row[2]),
			Low:      parseBoardFloat(row[3]),
			Close:    parseBoardFloat(row[4]),
			Volume:   parseBoardFloat(row[5]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("bybit returned no klines")
	}
	return out, nil
}

func fetchOKXKlines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	url := fmt.Sprintf(
		"https://www.okx.com/api/v5/market/candles?instId=%s&bar=%s&limit=%d",
		symbol, interval, limit,
	)
	var raw struct {
		Code string     `json:"code"`
		Msg  string     `json:"msg"`
		Data [][]string `json:"data"`
	}
	if err := getBoardJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("okx: %s", raw.Msg)
	}
	// OKX newest-first: [ts, o, h, l, c, vol, volCcy, volCcyQuote, confirm]
	out := make([]Candle, 0, len(raw.Data))
	for i := len(raw.Data) - 1; i >= 0; i-- {
		row := raw.Data[i]
		if len(row) < 6 {
			continue
		}
		ts := int64(parseBoardFloat(row[0]))
		out = append(out, Candle{
			OpenTime: ts,
			Open:     parseBoardFloat(row[1]),
			High:     parseBoardFloat(row[2]),
			Low:      parseBoardFloat(row[3]),
			Close:    parseBoardFloat(row[4]),
			Volume:   parseBoardFloat(row[5]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("okx returned no klines")
	}
	return out, nil
}

func fetchGateKlines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	url := fmt.Sprintf(
		"https://api.gateio.ws/api/v4/futures/usdt/candlesticks?contract=%s&interval=%s&limit=%d",
		symbol, interval, limit,
	)
	var raw []struct {
		T any `json:"t"`
		O any `json:"o"`
		H any `json:"h"`
		L any `json:"l"`
		C any `json:"c"`
		V any `json:"v"`
	}
	if err := getBoardJSONFirst(ctx, []string{
		url,
		strings.Replace(url, "api.gateio.ws", "api.gate.com", 1),
	}, &raw); err != nil {
		return nil, err
	}
	out := make([]Candle, 0, len(raw))
	for _, row := range raw {
		ts := anyToInt64(row.T)
		if ts > 0 && ts < 1e12 {
			ts *= 1000
		}
		out = append(out, Candle{
			OpenTime: ts,
			Open:     anyToFloat(row.O),
			High:     anyToFloat(row.H),
			Low:      anyToFloat(row.L),
			Close:    anyToFloat(row.C),
			Volume:   anyToFloat(row.V),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("gate returned no klines")
	}
	return out, nil
}

func fetchBitgetKlines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	url := fmt.Sprintf(
		"https://api.bitget.com/api/v2/mix/market/candles?symbol=%s&productType=USDT-FUTURES&granularity=%s&limit=%d",
		symbol, interval, limit,
	)
	var raw struct {
		Code string     `json:"code"`
		Msg  string     `json:"msg"`
		Data [][]string `json:"data"`
	}
	if err := getBoardJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	if raw.Code != "" && raw.Code != "00000" {
		return nil, fmt.Errorf("bitget: %s", raw.Msg)
	}
	out := make([]Candle, 0, len(raw.Data))
	for _, row := range raw.Data {
		if len(row) < 6 {
			continue
		}
		// Bitget often returns oldest-first: ts,o,h,l,c,baseVol,quoteVol
		out = append(out, Candle{
			OpenTime: int64(parseBoardFloat(row[0])),
			Open:     parseBoardFloat(row[1]),
			High:     parseBoardFloat(row[2]),
			Low:      parseBoardFloat(row[3]),
			Close:    parseBoardFloat(row[4]),
			Volume:   parseBoardFloat(row[5]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("bitget returned no klines")
	}
	return out, nil
}

func fetchKucoinKlines(ctx context.Context, symbol, interval string, limit int) ([]Candle, error) {
	// KuCoin futures: granularity in minutes; endAt optional.
	endAt := time.Now().Unix()
	url := fmt.Sprintf(
		"https://api-futures.kucoin.com/api/v1/kline/query?symbol=%s&granularity=%s&to=%d",
		symbol, interval, endAt,
	)
	var raw struct {
		Code string    `json:"code"`
		Msg  string    `json:"msg"`
		Data [][]any   `json:"data"`
	}
	if err := getBoardJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	if raw.Code != "" && raw.Code != "200000" {
		return nil, fmt.Errorf("kucoin: %s", raw.Msg)
	}
	out := make([]Candle, 0, len(raw.Data))
	for _, row := range raw.Data {
		if len(row) < 6 {
			continue
		}
		ts := anyToInt64(row[0])
		if ts < 1e12 {
			ts *= 1000
		}
		out = append(out, Candle{
			OpenTime: ts,
			Open:     anyToFloat(row[1]),
			High:     anyToFloat(row[2]),
			Low:      anyToFloat(row[3]),
			Close:    anyToFloat(row[4]),
			Volume:   anyToFloat(row[5]),
		})
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("kucoin returned no klines")
	}
	return out, nil
}

func anyToFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		return parseBoardFloat(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func anyToInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		return int64(parseBoardFloat(t))
	case int:
		return int64(t)
	case int64:
		return t
	default:
		return 0
	}
}
