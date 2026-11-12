package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"strand/internal/hook"
)

// BoardSymbol is a normalized public market-board row used by Markets / Chat.
type BoardSymbol struct {
	Symbol       string  `json:"symbol"`
	Display      string  `json:"display"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Exchange     string  `json:"exchange"`
	MarkPrice    float64 `json:"mark_price"`
	Change24hPct float64 `json:"change_24h_pct"`
	Volume24h    float64 `json:"volume_24h"`
	PrevDayPrice float64 `json:"prev_day_price,omitempty"`
}

type boardCacheEntry struct {
	at   time.Time
	rows []BoardSymbol
}

var (
	boardHTTPOnce sync.Once
	boardHTTP     *http.Client
	boardCacheMu  sync.Mutex
	boardCache    = map[string]boardCacheEntry{}
)

const boardCacheTTL = 20 * time.Second

func boardHTTPClient() *http.Client {
	boardHTTPOnce.Do(func() {
		client := &http.Client{Timeout: 25 * time.Second}
		hookRes := hook.HookExec[hook.SetHttpClientResult](hook.SET_HTTP_CLIENT, client)
		if hookRes != nil && hookRes.Error() == nil {
			client = hookRes.GetResult()
		}
		boardHTTP = client
	})
	return boardHTTP
}

func getBoardJSON(ctx context.Context, url string, dest any) error {
	// Cap each attempt so multi-host fallbacks can still fit the handler budget.
	attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "strand-markets/1.0")

	resp, err := boardHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 180 {
			snippet = snippet[:180] + "…"
		}
		return fmt.Errorf("%s returned %d: %s", url, resp.StatusCode, snippet)
	}
	return json.Unmarshal(body, dest)
}

func getBoardJSONFirst(ctx context.Context, urls []string, dest any) error {
	var errs []string
	for _, url := range urls {
		if err := getBoardJSON(ctx, url, dest); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		return nil
	}
	if len(errs) == 0 {
		return fmt.Errorf("no ticker urls provided")
	}
	return fmt.Errorf("%s", strings.Join(errs, " | "))
}

func parseBoardFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func displayFromSymbol(symbol, exchange string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	switch exchange {
	case "okx":
		// BTC-USDT-SWAP
		parts := strings.Split(s, "-")
		if len(parts) > 0 {
			return parts[0]
		}
	case "gate":
		// BTC_USDT
		if i := strings.Index(s, "_"); i > 0 {
			return s[:i]
		}
	default:
		for _, quote := range []string{"USDT", "USDC", "USD"} {
			if strings.HasSuffix(s, quote) && len(s) > len(quote) {
				return s[:len(s)-len(quote)]
			}
		}
	}
	return s
}

func finishBoard(exchange string, rows []BoardSymbol, limit int) []BoardSymbol {
	for i := range rows {
		rows[i].Exchange = exchange
		rows[i].Category = "crypto"
		if rows[i].Display == "" {
			rows[i].Display = displayFromSymbol(rows[i].Symbol, exchange)
		}
		if rows[i].Name == "" {
			rows[i].Name = rows[i].Display
		}
		if rows[i].PrevDayPrice <= 0 && rows[i].MarkPrice > 0 && rows[i].Change24hPct != 0 {
			den := 1 + rows[i].Change24hPct/100
			if den != 0 {
				rows[i].PrevDayPrice = rows[i].MarkPrice / den
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Volume24h > rows[j].Volume24h
	})
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

// FetchCEXBoard loads a live USDT-perp board for the given CEX.
// Results are cached briefly to keep Markets refresh cheap.
func FetchCEXBoard(ctx context.Context, exchange string, limit int) ([]BoardSymbol, error) {
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}

	boardCacheMu.Lock()
	if entry, ok := boardCache[exchange]; ok && time.Since(entry.at) < boardCacheTTL {
		cached := append([]BoardSymbol(nil), entry.rows...)
		boardCacheMu.Unlock()
		if limit < len(cached) {
			cached = cached[:limit]
		}
		return cached, nil
	}
	boardCacheMu.Unlock()

	var (
		rows []BoardSymbol
		err  error
	)
	switch exchange {
	case "binance":
		rows, err = fetchBinanceBoard(ctx)
	case "bybit":
		rows, err = fetchBybitBoard(ctx)
	case "okx":
		rows, err = fetchOKXBoard(ctx)
	case "gate":
		rows, err = fetchGateBoard(ctx)
	case "bitget":
		rows, err = fetchBitgetBoard(ctx)
	case "kucoin":
		rows, err = fetchKucoinBoard(ctx)
	default:
		return nil, fmt.Errorf("live board unsupported for exchange %s", exchange)
	}
	if err != nil {
		return nil, err
	}
	rows = finishBoard(exchange, rows, limit)

	boardCacheMu.Lock()
	boardCache[exchange] = boardCacheEntry{at: time.Now(), rows: append([]BoardSymbol(nil), rows...)}
	boardCacheMu.Unlock()
	return rows, nil
}

func fetchBinanceBoard(ctx context.Context) ([]BoardSymbol, error) {
	var raw []struct {
		Symbol             string `json:"symbol"`
		LastPrice          string `json:"lastPrice"`
		PriceChangePercent string `json:"priceChangePercent"`
		QuoteVolume        string `json:"quoteVolume"`
		Volume             string `json:"volume"`
		PrevClosePrice     string `json:"prevClosePrice"`
	}
	if err := getBoardJSONFirst(ctx, []string{
		"https://fapi.binance.com/fapi/v1/ticker/24hr",
		"https://fapi1.binance.com/fapi/v1/ticker/24hr",
		"https://fapi2.binance.com/fapi/v1/ticker/24hr",
	}, &raw); err != nil {
		return nil, err
	}
	out := make([]BoardSymbol, 0, len(raw))
	for _, item := range raw {
		sym := strings.ToUpper(item.Symbol)
		// Perps: BTCUSDT. Delivery: BTCUSDT_250328.
		if !strings.HasSuffix(sym, "USDT") || strings.Contains(sym, "_") {
			continue
		}
		price := parseBoardFloat(item.LastPrice)
		if price <= 0 {
			continue
		}
		out = append(out, BoardSymbol{
			Symbol:       sym,
			MarkPrice:    price,
			Change24hPct: parseBoardFloat(item.PriceChangePercent),
			Volume24h:    parseBoardFloat(item.QuoteVolume),
			PrevDayPrice: parseBoardFloat(item.PrevClosePrice),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("binance returned no USDT perpetual tickers")
	}
	return out, nil
}

func fetchBybitBoard(ctx context.Context) ([]BoardSymbol, error) {
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Symbol        string `json:"symbol"`
				LastPrice     string `json:"lastPrice"`
				Price24hPcnt  string `json:"price24hPcnt"`
				Turnover24h   string `json:"turnover24h"`
				PrevPrice24h  string `json:"prevPrice24h"`
				QuoteCoin     string `json:"quoteCoin"`
				Status        string `json:"status"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := getBoardJSONFirst(ctx, []string{
		"https://api.bybit.com/v5/market/tickers?category=linear",
		"https://api.bytick.com/v5/market/tickers?category=linear",
	}, &raw); err != nil {
		return nil, err
	}
	if raw.RetCode != 0 {
		return nil, fmt.Errorf("bybit: %s", raw.RetMsg)
	}
	out := make([]BoardSymbol, 0, len(raw.Result.List))
	for _, item := range raw.Result.List {
		sym := strings.ToUpper(item.Symbol)
		if !strings.HasSuffix(sym, "USDT") || strings.Contains(sym, "-") {
			continue
		}
		if item.Status != "" && !strings.EqualFold(item.Status, "Trading") {
			continue
		}
		price := parseBoardFloat(item.LastPrice)
		if price <= 0 {
			continue
		}
		// Bybit price24hPcnt is a fraction (0.01 = 1%).
		change := parseBoardFloat(item.Price24hPcnt) * 100
		out = append(out, BoardSymbol{
			Symbol:       sym,
			MarkPrice:    price,
			Change24hPct: change,
			Volume24h:    parseBoardFloat(item.Turnover24h),
			PrevDayPrice: parseBoardFloat(item.PrevPrice24h),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("bybit returned no USDT linear tickers")
	}
	return out, nil
}

func fetchOKXBoard(ctx context.Context) ([]BoardSymbol, error) {
	var raw struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			InstID    string `json:"instId"`
			Last      string `json:"last"`
			Open24h   string `json:"open24h"`
			VolCcy24h string `json:"volCcy24h"`
			Vol24h    string `json:"vol24h"`
		} `json:"data"`
	}
	if err := getBoardJSON(ctx, "https://www.okx.com/api/v5/market/tickers?instType=SWAP", &raw); err != nil {
		return nil, err
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("okx: %s", raw.Msg)
	}
	out := make([]BoardSymbol, 0, len(raw.Data))
	for _, item := range raw.Data {
		sym := strings.ToUpper(item.InstID)
		if !strings.HasSuffix(sym, "-USDT-SWAP") {
			continue
		}
		price := parseBoardFloat(item.Last)
		if price <= 0 {
			continue
		}
		open := parseBoardFloat(item.Open24h)
		change := 0.0
		if open > 0 {
			change = (price - open) / open * 100
		}
		vol := parseBoardFloat(item.VolCcy24h)
		if vol <= 0 {
			vol = parseBoardFloat(item.Vol24h) * price
		}
		out = append(out, BoardSymbol{
			Symbol:       sym,
			Display:      displayFromSymbol(sym, "okx"),
			MarkPrice:    price,
			Change24hPct: change,
			Volume24h:    vol,
			PrevDayPrice: open,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("okx returned no USDT swap tickers")
	}
	return out, nil
}

func fetchGateBoard(ctx context.Context) ([]BoardSymbol, error) {
	var raw []struct {
		Contract         string `json:"contract"`
		Last             string `json:"last"`
		MarkPrice        string `json:"mark_price"`
		ChangePercentage string `json:"change_percentage"`
		Volume24hQuote   string `json:"volume_24h_quote"`
		Volume24hSettle  string `json:"volume_24h_settle"`
	}
	if err := getBoardJSONFirst(ctx, []string{
		"https://api.gateio.ws/api/v4/futures/usdt/tickers",
		"https://api.gate.com/api/v4/futures/usdt/tickers",
	}, &raw); err != nil {
		return nil, err
	}
	out := make([]BoardSymbol, 0, len(raw))
	for _, item := range raw {
		sym := strings.ToUpper(item.Contract)
		if !strings.HasSuffix(sym, "_USDT") {
			continue
		}
		price := parseBoardFloat(item.MarkPrice)
		if price <= 0 {
			price = parseBoardFloat(item.Last)
		}
		if price <= 0 {
			continue
		}
		vol := parseBoardFloat(item.Volume24hQuote)
		if vol <= 0 {
			vol = parseBoardFloat(item.Volume24hSettle)
		}
		out = append(out, BoardSymbol{
			Symbol:       sym,
			Display:      displayFromSymbol(sym, "gate"),
			MarkPrice:    price,
			Change24hPct: parseBoardFloat(item.ChangePercentage),
			Volume24h:    vol,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("gate returned no USDT futures tickers")
	}
	return out, nil
}

func fetchBitgetBoard(ctx context.Context) ([]BoardSymbol, error) {
	var raw struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Symbol       string `json:"symbol"`
			LastPr       string `json:"lastPr"`
			Change24h    string `json:"change24h"`
			QuoteVolume  string `json:"quoteVolume"`
			UsdtVolume   string `json:"usdtVolume"`
			Open24h      string `json:"open24h"`
		} `json:"data"`
	}
	if err := getBoardJSON(ctx, "https://api.bitget.com/api/v2/mix/market/tickers?productType=USDT-FUTURES", &raw); err != nil {
		return nil, err
	}
	if raw.Code != "" && raw.Code != "00000" {
		return nil, fmt.Errorf("bitget: %s", raw.Msg)
	}
	out := make([]BoardSymbol, 0, len(raw.Data))
	for _, item := range raw.Data {
		sym := strings.ToUpper(item.Symbol)
		if !strings.HasSuffix(sym, "USDT") {
			continue
		}
		price := parseBoardFloat(item.LastPr)
		if price <= 0 {
			continue
		}
		// Bitget change24h is typically a fraction.
		change := parseBoardFloat(item.Change24h)
		if change != 0 && change > -1 && change < 1 {
			change *= 100
		}
		vol := parseBoardFloat(item.QuoteVolume)
		if vol <= 0 {
			vol = parseBoardFloat(item.UsdtVolume)
		}
		out = append(out, BoardSymbol{
			Symbol:       sym,
			MarkPrice:    price,
			Change24hPct: change,
			Volume24h:    vol,
			PrevDayPrice: parseBoardFloat(item.Open24h),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("bitget returned no USDT futures tickers")
	}
	return out, nil
}

func fetchKucoinBoard(ctx context.Context) ([]BoardSymbol, error) {
	var raw struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Ticker []struct {
				Symbol       string `json:"symbol"`
				LastTradePrice string `json:"lastTradePrice"`
				PriceChgPct  string `json:"priceChgPct"`
				Turnover     string `json:"turnover"`
			} `json:"ticker"`
		} `json:"data"`
	}
	if err := getBoardJSON(ctx, "https://api-futures.kucoin.com/api/v1/allTickers", &raw); err != nil {
		return nil, err
	}
	if raw.Code != "" && raw.Code != "200000" {
		return nil, fmt.Errorf("kucoin: %s", raw.Msg)
	}
	out := make([]BoardSymbol, 0, len(raw.Data.Ticker))
	for _, item := range raw.Data.Ticker {
		sym := strings.ToUpper(item.Symbol)
		// XBTUSDTM / ETHUSDTM etc.
		if !strings.HasSuffix(sym, "USDTM") {
			continue
		}
		price := parseBoardFloat(item.LastTradePrice)
		if price <= 0 {
			continue
		}
		display := strings.TrimSuffix(sym, "USDTM")
		if display == "XBT" {
			display = "BTC"
		}
		change := parseBoardFloat(item.PriceChgPct)
		if change != 0 && change > -1 && change < 1 {
			change *= 100
		}
		out = append(out, BoardSymbol{
			Symbol:       sym,
			Display:      display,
			MarkPrice:    price,
			Change24hPct: change,
			Volume24h:    parseBoardFloat(item.Turnover),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("kucoin returned no USDTM tickers")
	}
	return out, nil
}

// SupportsLiveCEXBoard reports whether FetchCEXBoard can serve the venue.
func SupportsLiveCEXBoard(exchange string) bool {
	switch strings.ToLower(strings.TrimSpace(exchange)) {
	case "binance", "bybit", "okx", "gate", "bitget", "kucoin":
		return true
	default:
		return false
	}
}

// LogCEXBoardError is a tiny helper so handlers stay quiet on happy path.
func LogCEXBoardError(exchange string, err error) {
	if err == nil {
		return
	}
	log.Printf("cex board %s: %v", exchange, err)
}
