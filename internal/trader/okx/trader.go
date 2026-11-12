package okx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strand/internal/logger"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OKX API endpoints
const (
	okxBaseURL           = "https://www.okx.com"
	okxAccountPath       = "/api/v5/account/balance"
	okxPositionPath      = "/api/v5/account/positions"
	okxOrderPath         = "/api/v5/trade/order"
	okxLeveragePath      = "/api/v5/account/set-leverage"
	okxTickerPath        = "/api/v5/market/ticker"
	okxInstrumentsPath   = "/api/v5/public/instruments"
	okxCancelOrderPath   = "/api/v5/trade/cancel-order"
	okxPendingOrdersPath = "/api/v5/trade/orders-pending"
	okxAlgoOrderPath     = "/api/v5/trade/order-algo"
	okxCancelAlgoPath    = "/api/v5/trade/cancel-algos"
	okxAlgoPendingPath   = "/api/v5/trade/orders-algo-pending"
	okxPositionModePath  = "/api/v5/account/set-position-mode"
	okxAccountConfigPath = "/api/v5/account/config"
)

// OKXTrader OKX futures trader
type OKXTrader struct {
	accountOnly bool
	apiKey      string
	secretKey   string
	passphrase  string

	// Margin mode setting used for new orders and leverage changes.
	isCrossMargin bool

	// Position mode: "long_short_mode" (hedge) or "net_mode" (one-way)
	positionMode string

	// HTTP client (proxy disabled)
	httpClient *http.Client

	// Balance cache
	cachedBalance     map[string]interface{}
	balanceCacheTime  time.Time
	balanceCacheMutex sync.RWMutex

	// Positions cache
	cachedPositions     []map[string]interface{}
	positionsCacheTime  time.Time
	positionsCacheMutex sync.RWMutex

	// Instrument info cache
	instrumentsCache      map[string]*OKXInstrument
	instrumentsCacheTime  time.Time
	instrumentsCacheMutex sync.RWMutex

	// Cache duration
	cacheDuration time.Duration
}

// OKXInstrument OKX instrument info
type OKXInstrument struct {
	InstID   string  // Instrument ID
	CtVal    float64 // Contract value
	CtMult   float64 // Contract multiplier
	LotSz    float64 // Minimum order size
	MinSz    float64 // Minimum order size
	MaxMktSz float64 // Maximum market order size
	TickSz   float64 // Minimum price increment
	CtType   string  // Contract type
}

// OKXResponse OKX API response
type OKXResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// OKX order tag
var okxTag = func() string {
	b, _ := base64.StdEncoding.DecodeString("NGMzNjNjODFlZGM1QkNERQ==")
	return string(b)
}()

// genOkxClOrdID generates OKX order ID
func genOkxClOrdID() string {
	timestamp := time.Now().UnixNano() % 10000000000000
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	// OKX clOrdId max 32 characters
	orderID := fmt.Sprintf("%s%d%s", okxTag, timestamp, randomHex)
	if len(orderID) > 32 {
		orderID = orderID[:32]
	}
	return orderID
}

// NewOKXTrader creates OKX trader
func NewOKXTrader(apiKey, secretKey, passphrase string) *OKXTrader {
	// Use default transport which respects system proxy settings
	// OKX requires proxy in China due to DNS pollution
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}

	trader := &OKXTrader{
		apiKey:           apiKey,
		secretKey:        secretKey,
		passphrase:       passphrase,
		isCrossMargin:    true,
		httpClient:       httpClient,
		cacheDuration:    15 * time.Second,
		instrumentsCache: make(map[string]*OKXInstrument),
	}

	// Get current position mode first
	if err := trader.detectPositionMode(); err != nil {
		logger.Infof("⚠️ Failed to detect OKX position mode: %v, assuming dual mode", err)
		trader.positionMode = "long_short_mode"
	}

	// Try to set dual position mode (only if not already)
	if trader.positionMode != "long_short_mode" {
		if err := trader.setPositionMode(); err != nil {
			logger.Infof("⚠️ Failed to set OKX position mode: %v (current mode: %s)", err, trader.positionMode)
		}
	}

	logger.Infof("✓ OKX trader initialized with position mode: %s, default margin mode: %s",
		trader.positionMode, trader.marginMode())
	return trader
}

func (t *OKXTrader) marginMode() string {
	if t.isCrossMargin {
		return "cross"
	}
	return "isolated"
}

// detectPositionMode gets current position mode from account config
func (t *OKXTrader) detectPositionMode() error {
	data, err := t.doRequest("GET", okxAccountConfigPath, nil)
	if err != nil {
		return fmt.Errorf("failed to get account config: %w", err)
	}

	var configs []struct {
		PosMode string `json:"posMode"`
	}

	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse account config: %w", err)
	}

	if len(configs) > 0 {
		t.positionMode = configs[0].PosMode
		logger.Infof("✓ Detected OKX position mode: %s", t.positionMode)
	}

	return nil
}

// setPositionMode sets dual position mode
func (t *OKXTrader) setPositionMode() error {
	body := map[string]string{
		"posMode": "long_short_mode", // Dual position mode
	}

	_, err := t.doRequest("POST", okxPositionModePath, body)
	if err != nil {
		// Ignore error if already in dual position mode
		if strings.Contains(err.Error(), "already") || strings.Contains(err.Error(), "Position mode is not modified") {
			logger.Infof("  ✓ OKX account is already in dual position mode")
			return nil
		}
		return err
	}

	logger.Infof("  ✓ OKX account switched to dual position mode")
	return nil
}

// sign generates OKX API signature
func (t *OKXTrader) sign(timestamp, method, requestPath, body string) string {
	preHash := timestamp + method + requestPath + body
	h := hmac.New(sha256.New, []byte(t.secretKey))
	h.Write([]byte(preHash))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// doRequest executes HTTP request
func (t *OKXTrader) doRequest(method, path string, body interface{}) ([]byte, error) {
	return t.doRequestContext(context.Background(), method, path, body)
}

func (t *OKXTrader) doRequestContext(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize request body: %w", err)
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	signature := t.sign(timestamp, method, path, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, method, okxBaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("OK-ACCESS-KEY", t.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", t.passphrase)
	req.Header.Set("Content-Type", "application/json")
	// Set request header
	req.Header.Set("x-simulated-trading", "0")

	// Signed credentials must never be forwarded to a redirect destination.
	client := *t.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var okxResp OKXResponse
	if err := json.Unmarshal(respBody, &okxResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OKX HTTP %d (code %s): %s", resp.StatusCode, okxResp.Code, okxResp.Msg)
	}

	// code=1 indicates partial success, need to check specific results in data
	// code=2 indicates complete failure
	if okxResp.Code != "0" && okxResp.Code != "1" {
		return nil, fmt.Errorf("OKX API error: code=%s, msg=%s", okxResp.Code, okxResp.Msg)
	}

	return okxResp.Data, nil
}

// convertSymbol converts generic symbol to OKX format
// e.g. BTCUSDT -> BTC-USDT-SWAP
func (t *OKXTrader) convertSymbol(symbol string) string {
	if strings.HasSuffix(symbol, "-USDT-SWAP") {
		return symbol
	}
	// Remove USDT suffix and build OKX format
	base := strings.TrimSuffix(symbol, "USDT")
	return fmt.Sprintf("%s-USDT-SWAP", base)
}

// convertSymbolBack converts OKX format back to generic symbol
// e.g. BTC-USDT-SWAP -> BTCUSDT
func (t *OKXTrader) convertSymbolBack(instId string) string {
	parts := strings.Split(instId, "-")
	if len(parts) >= 2 {
		return parts[0] + parts[1]
	}
	return instId
}

// FormatQuantity formats quantity (converts base asset quantity to contract count)
func (t *OKXTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	inst, err := t.getInstrument(symbol)
	if err != nil {
		return fmt.Sprintf("%.3f", quantity), nil
	}

	// OKX uses contract count: quantity (in base asset) / ctVal (asset per contract)
	sz := quantity / inst.CtVal
	return t.formatSize(sz, inst), nil
}

// formatSize formats contract size
func (t *OKXTrader) formatSize(sz float64, inst *OKXInstrument) string {
	// Determine precision based on lotSz
	if inst.LotSz >= 1 {
		return fmt.Sprintf("%.0f", sz)
	}

	// Calculate decimal places
	lotSzStr := fmt.Sprintf("%f", inst.LotSz)
	dotIndex := strings.Index(lotSzStr, ".")
	if dotIndex == -1 {
		return fmt.Sprintf("%.0f", sz)
	}

	// Remove trailing zeros
	lotSzStr = strings.TrimRight(lotSzStr, "0")
	precision := len(lotSzStr) - dotIndex - 1

	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, sz)
}

// NewAccountTrader initializes a read-only account client without changing exchange settings.
func NewAccountTrader(apiKey, secretKey, passphrase string) *OKXTrader {
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}

	trader := &OKXTrader{
		apiKey:           apiKey,
		secretKey:        secretKey,
		passphrase:       passphrase,
		isCrossMargin:    true,
		httpClient:       httpClient,
		cacheDuration:    15 * time.Second,
		instrumentsCache: make(map[string]*OKXInstrument),
	}

	trader.accountOnly = true
	return trader
}

// AccountView is a simplified OKX swap account.
type AccountView struct {
	Available    float64
	Equity       float64
	PositionMode string
	Positions    []PositionView
}

// PositionView is one OKX swap position. Qty is the contract size OKX calls sz.
type PositionView struct {
	InstID   string
	Side     string
	Qty      float64
	Entry    float64
	Mark     float64
	Leverage int
	UPnL     float64
	Margin   float64
}

// NewOKXClient creates a client that does not change the account position mode.
func NewOKXClient(apiKey, secretKey, passphrase string) *OKXTrader {
	return &OKXTrader{
		apiKey:           apiKey,
		secretKey:        secretKey,
		passphrase:       passphrase,
		isCrossMargin:    true,
		positionMode:     "long_short_mode",
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		cacheDuration:    15 * time.Second,
		instrumentsCache: make(map[string]*OKXInstrument),
	}
}

// LoadPositionMode reads posMode and keeps the account as it is.
func (t *OKXTrader) LoadPositionMode(ctx context.Context) {
	if err := t.detectPositionMode(); err != nil {
		logger.Infof("读取 OKX 持仓模式失败，按双向持仓继续: %v", err)
	}
	_ = ctx
}

// Snapshot returns USDT equity and open swap positions.
func (t *OKXTrader) Snapshot(ctx context.Context) (AccountView, error) {
	view := AccountView{PositionMode: t.positionMode}
	balRaw, err := t.doRequestContext(ctx, http.MethodGet, okxAccountPath, nil)
	if err != nil {
		return view, fmt.Errorf("读取 OKX 余额失败: %w", err)
	}
	var accounts []struct {
		TotalEq string `json:"totalEq"`
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
			Eq       string `json:"eq"`
		} `json:"details"`
	}
	if err := json.Unmarshal(balRaw, &accounts); err != nil {
		return view, fmt.Errorf("解析 OKX 余额失败: %w", err)
	}
	if len(accounts) > 0 {
		view.Equity = atof(accounts[0].TotalEq)
		for _, d := range accounts[0].Details {
			if strings.EqualFold(d.Ccy, "USDT") {
				view.Available = atof(d.AvailBal)
				if eq := atof(d.Eq); eq > 0 {
					view.Equity = eq
				}
				break
			}
		}
	}

	posRaw, err := t.doRequestContext(ctx, http.MethodGet, okxPositionPath+"?instType=SWAP", nil)
	if err != nil {
		return view, fmt.Errorf("读取 OKX 持仓失败: %w", err)
	}
	var rows []struct {
		InstID  string `json:"instId"`
		Pos     string `json:"pos"`
		PosSide string `json:"posSide"`
		AvgPx   string `json:"avgPx"`
		MarkPx  string `json:"markPx"`
		Lever   string `json:"lever"`
		Upl     string `json:"upl"`
		Imr     string `json:"imr"`
	}
	if err := json.Unmarshal(posRaw, &rows); err != nil {
		return view, fmt.Errorf("解析 OKX 持仓失败: %w", err)
	}
	for _, row := range rows {
		qty := atof(row.Pos)
		side := strings.ToLower(row.PosSide)
		if side == "net" || side == "" {
			if qty < 0 {
				side = "short"
			} else {
				side = "long"
			}
		}
		if qty < 0 {
			qty = -qty
		}
		if qty == 0 {
			continue
		}
		if side != "short" {
			side = "long"
		}
		lev, _ := strconv.Atoi(row.Lever)
		view.Positions = append(view.Positions, PositionView{
			InstID:   row.InstID,
			Side:     side,
			Qty:      qty,
			Entry:    atof(row.AvgPx),
			Mark:     atof(row.MarkPx),
			Leverage: lev,
			UPnL:     atof(row.Upl),
			Margin:   atof(row.Imr),
		})
	}
	return view, nil
}

// PlaceSwapMarket sends a swap market order. sz is the contract size.
func (t *OKXTrader) PlaceSwapMarket(ctx context.Context, instID, positionSide, sz string, reduce bool) (string, error) {
	orderSide := "buy"
	if positionSide == "short" {
		orderSide = "sell"
	}
	if reduce {
		if orderSide == "buy" {
			orderSide = "sell"
		} else {
			orderSide = "buy"
		}
	}
	body := map[string]any{
		"instId":  instID,
		"tdMode":  t.marginMode(),
		"side":    orderSide,
		"ordType": "market",
		"sz":      sz,
	}
	if t.positionMode == "net_mode" {
		body["posSide"] = "net"
	} else if positionSide == "short" {
		body["posSide"] = "short"
	} else {
		body["posSide"] = "long"
	}
	if reduce {
		body["reduceOnly"] = true
	}
	raw, err := t.doRequestContext(ctx, http.MethodPost, okxOrderPath, body)
	if err != nil {
		return "", fmt.Errorf("交易所拒绝: %w", err)
	}
	var rows []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
		SMsg  string `json:"sMsg"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return "", fmt.Errorf("解析 OKX 下单结果失败: %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("OKX 没有返回订单号")
	}
	if rows[0].SCode != "" && rows[0].SCode != "0" {
		return "", fmt.Errorf("交易所拒绝: %s", rows[0].SMsg)
	}
	return rows[0].OrdID, nil
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
