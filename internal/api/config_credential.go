package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"strand/internal/mcp"
	"strand/internal/security"
	"strand/internal/store"
	"strand/internal/trader/aster"
	"strand/internal/trader/binance"
	"strand/internal/trader/bitget"
	"strand/internal/trader/bybit"
	"strand/internal/trader/gate"
	"strand/internal/trader/hyperliquid"
	"strand/internal/trader/indodax"
	"strand/internal/trader/kucoin"
	"strand/internal/trader/lighter"
	"strand/internal/trader/okx"
	"strand/internal/trader/types"
)

type modelCredentialInput struct {
	Provider        string
	APIKey          string
	CustomAPIURL    string
	CustomModelName string
}

type exchangeCredentialInput struct {
	ExchangeType            string
	APIKey                  string
	SecretKey               string
	Passphrase              string
	Testnet                 bool
	HyperliquidWalletAddr   string
	HyperliquidUnifiedAcct  bool
	AsterUser               string
	AsterSigner             string
	AsterPrivateKey         string
	LighterWalletAddr       string
	LighterPrivateKey       string
	LighterAPIKeyPrivateKey string
	LighterAPIKeyIndex      int
	UserID                  string
}

func resolveAIEndpoint(provider, customURL, customModel string) (baseURL, model string) {
	return mcp.ResolveEndpoint(provider, customURL, customModel)
}

func buildCredentialAIClient(in modelCredentialInput) (mcp.AIClient, error) {
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	if provider == "" {
		return nil, fmt.Errorf("provider required")
	}
	apiKey := strings.TrimSpace(in.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("API key / wallet key required")
	}

	var client mcp.AIClient
	if provider == "custom" {
		client = mcp.New()
	} else {
		client = mcp.NewAIClientByProvider(provider)
	}
	if client == nil {
		client = mcp.New()
	}

	baseURL, model := resolveAIEndpoint(provider, in.CustomAPIURL, in.CustomModelName)
	switch provider {
	case "claw402":
		client.SetAPIKey(apiKey, "", model)
	default:
		client.SetAPIKey(apiKey, baseURL, model)
	}
	if setter, ok := client.(interface{ SetTimeout(time.Duration) }); ok {
		// Credential probes can be slow behind an outbound proxy.
		setter.SetTimeout(45 * time.Second)
	}
	return client, nil
}

func openAICompatibleModelsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	// Accept either ".../v1" or a bare host; avoid double /models.
	if strings.HasSuffix(base, "/models") {
		return base
	}
	return base + "/models"
}

func providerUsesOpenAICompatibleProbe(provider string) bool {
	switch provider {
	case "openai", "deepseek", "qwen", "grok", "gemini", "kimi", "minimax", "custom":
		return true
	default:
		return false
	}
}

// probeOpenAICompatibleAuth hits GET /models — validates the key without a
// paid chat completion and fails fast on network/proxy issues.
func probeOpenAICompatibleAuth(baseURL, apiKey string) error {
	endpoint := openAICompatibleModelsURL(baseURL)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := security.SafeHTTPClient(45 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return annotateAINetworkError(err, baseURL)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("API key rejected (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	default:
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("provider returned %d: %s", resp.StatusCode, msg)
	}
}

func annotateAINetworkError(err error, baseURL string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	hint := ""
	switch {
	case strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "i/o timeout"):
		hint = "network timeout reaching the AI endpoint — if you are behind a firewall, start Strand with HTTPS_PROXY/HTTP_PROXY, or set a reachable Custom Base URL (OpenAI-compatible relay)"
	case strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "dns") ||
		strings.Contains(lower, "lookup"):
		hint = "DNS could not resolve the AI host — configure HTTPS_PROXY or a Custom Base URL that your network can reach"
	case strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "network is unreachable"):
		hint = "cannot connect to the AI host — check proxy settings or Custom Base URL"
	}
	target := strings.TrimSpace(baseURL)
	if target == "" {
		target = "https://api.openai.com/v1"
	}
	if hint != "" {
		return fmt.Errorf("%s (endpoint %s): %w", hint, target, err)
	}
	return fmt.Errorf("endpoint %s: %w", target, err)
}

// testModelCredentials verifies credentials without persisting them.
// Claw402 only validates the EVM private key (users may save and fund later).
// OpenAI-compatible providers use GET /models; others issue a tiny completion.
func testModelCredentials(in modelCredentialInput) error {
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	apiKey := strings.TrimSpace(in.APIKey)
	if apiKey == "" {
		return fmt.Errorf("API key / wallet key required")
	}
	if provider == "claw402" {
		if _, err := addressFromPrivateKey(apiKey); err != nil {
			return fmt.Errorf("invalid Claw402 wallet key: %w", err)
		}
		return nil
	}

	baseURL, _ := resolveAIEndpoint(provider, in.CustomAPIURL, in.CustomModelName)
	if providerUsesOpenAICompatibleProbe(provider) {
		if err := probeOpenAICompatibleAuth(baseURL, apiKey); err != nil {
			return fmt.Errorf("AI connection failed: %w", err)
		}
		return nil
	}

	client, err := buildCredentialAIClient(in)
	if err != nil {
		return err
	}
	_, err = client.CallWithMessages(
		"You are a connection probe. Reply with exactly: ok",
		"ping",
	)
	if err != nil {
		return fmt.Errorf("AI connection failed: %w", annotateAINetworkError(err, baseURL))
	}
	return nil
}

func buildCredentialExchangeTrader(in exchangeCredentialInput) (types.Trader, error) {
	exType := strings.ToLower(strings.TrimSpace(in.ExchangeType))
	if exType == "" {
		return nil, fmt.Errorf("exchange_type required")
	}

	switch exType {
	case "binance":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" {
			return nil, fmt.Errorf("Binance API key and secret key required")
		}
		return binance.NewFuturesTrader(in.APIKey, in.SecretKey, in.UserID), nil
	case "bybit":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" {
			return nil, fmt.Errorf("Bybit API key and secret key required")
		}
		return bybit.NewBybitTrader(in.APIKey, in.SecretKey), nil
	case "okx":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" || strings.TrimSpace(in.Passphrase) == "" {
			return nil, fmt.Errorf("OKX API key, secret key and passphrase required")
		}
		return okx.NewOKXClient(in.APIKey, in.SecretKey, in.Passphrase), nil
	case "bitget":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" || strings.TrimSpace(in.Passphrase) == "" {
			return nil, fmt.Errorf("Bitget API key, secret key and passphrase required")
		}
		return bitget.NewBitgetTrader(in.APIKey, in.SecretKey, in.Passphrase), nil
	case "gate":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" {
			return nil, fmt.Errorf("Gate API key and secret key required")
		}
		return gate.NewGateTrader(in.APIKey, in.SecretKey), nil
	case "kucoin":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" || strings.TrimSpace(in.Passphrase) == "" {
			return nil, fmt.Errorf("KuCoin API key, secret key and passphrase required")
		}
		return kucoin.NewKuCoinTrader(in.APIKey, in.SecretKey, in.Passphrase), nil
	case "indodax":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.SecretKey) == "" {
			return nil, fmt.Errorf("Indodax API key and secret key required")
		}
		return indodax.NewIndodaxTrader(in.APIKey, in.SecretKey), nil
	case "hyperliquid":
		if strings.TrimSpace(in.APIKey) == "" || strings.TrimSpace(in.HyperliquidWalletAddr) == "" {
			return nil, fmt.Errorf("Hyperliquid agent key and wallet address required")
		}
		t, err := hyperliquid.NewHyperliquidTrader(
			in.APIKey,
			in.HyperliquidWalletAddr,
			in.Testnet,
			in.HyperliquidUnifiedAcct,
		)
		if err != nil {
			return nil, err
		}
		return t, nil
	case "aster":
		if strings.TrimSpace(in.AsterUser) == "" || strings.TrimSpace(in.AsterSigner) == "" || strings.TrimSpace(in.AsterPrivateKey) == "" {
			return nil, fmt.Errorf("Aster user, signer and private key required")
		}
		t, err := aster.NewAsterTrader(in.AsterUser, in.AsterSigner, in.AsterPrivateKey)
		if err != nil {
			return nil, err
		}
		return t, nil
	case "lighter":
		if strings.TrimSpace(in.LighterWalletAddr) == "" || strings.TrimSpace(in.LighterAPIKeyPrivateKey) == "" {
			return nil, fmt.Errorf("Lighter wallet address and API key private key required")
		}
		t, err := lighter.NewLighterTraderV2(
			in.LighterWalletAddr,
			in.LighterAPIKeyPrivateKey,
			in.LighterAPIKeyIndex,
			in.Testnet,
		)
		if err != nil {
			return nil, err
		}
		return t, nil
	default:
		return nil, fmt.Errorf("unsupported exchange type: %s", exType)
	}
}

func testExchangeCredentials(in exchangeCredentialInput) error {
	tr, err := buildCredentialExchangeTrader(in)
	if err != nil {
		return err
	}
	if _, err := tr.GetBalance(); err != nil {
		return fmt.Errorf("exchange connection failed: %w", err)
	}
	return nil
}

func exchangeCredentialFromStore(userID string, ex *store.Exchange) exchangeCredentialInput {
	if ex == nil {
		return exchangeCredentialInput{UserID: userID}
	}
	return exchangeCredentialInput{
		ExchangeType:            ex.ExchangeType,
		APIKey:                  string(ex.APIKey),
		SecretKey:               string(ex.SecretKey),
		Passphrase:              string(ex.Passphrase),
		Testnet:                 ex.Testnet,
		HyperliquidWalletAddr:   ex.HyperliquidWalletAddr,
		HyperliquidUnifiedAcct:  ex.HyperliquidUnifiedAcct,
		AsterUser:               ex.AsterUser,
		AsterSigner:             ex.AsterSigner,
		AsterPrivateKey:         string(ex.AsterPrivateKey),
		LighterWalletAddr:       ex.LighterWalletAddr,
		LighterPrivateKey:       string(ex.LighterPrivateKey),
		LighterAPIKeyPrivateKey: string(ex.LighterAPIKeyPrivateKey),
		LighterAPIKeyIndex:      ex.LighterAPIKeyIndex,
		UserID:                  userID,
	}
}

func exchangeHasCredentials(ex *store.Exchange) bool {
	if ex == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(ex.ExchangeType)) {
	case "hyperliquid":
		return strings.TrimSpace(string(ex.APIKey)) != "" &&
			strings.TrimSpace(ex.HyperliquidWalletAddr) != ""
	case "aster":
		return strings.TrimSpace(ex.AsterUser) != "" &&
			strings.TrimSpace(ex.AsterSigner) != "" &&
			strings.TrimSpace(string(ex.AsterPrivateKey)) != ""
	case "lighter":
		return strings.TrimSpace(ex.LighterWalletAddr) != "" &&
			strings.TrimSpace(string(ex.LighterAPIKeyPrivateKey)) != ""
	case "okx", "bitget", "kucoin":
		return strings.TrimSpace(string(ex.APIKey)) != "" &&
			strings.TrimSpace(string(ex.SecretKey)) != "" &&
			strings.TrimSpace(string(ex.Passphrase)) != ""
	default:
		return strings.TrimSpace(string(ex.APIKey)) != "" &&
			strings.TrimSpace(string(ex.SecretKey)) != ""
	}
}

func exchangeAsset(exchangeType string) string {
	switch strings.ToLower(strings.TrimSpace(exchangeType)) {
	case "hyperliquid", "aster", "lighter":
		return "USDC"
	case "indodax":
		return "IDR"
	default:
		return "USDT"
	}
}

func floatFromBalanceMap(bal map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if bal == nil {
			return 0, false
		}
		raw, ok := bal[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err == nil {
				return f, true
			}
		}
	}
	return 0, false
}
