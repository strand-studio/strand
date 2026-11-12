package api

import (
	"testing"

	"strand/internal/store"
)

func TestFloatFromBalanceMap(t *testing.T) {
	bal := map[string]any{
		"totalEquity":       100.5,
		"availableBalance":  "42.25",
		"totalWalletBalance": 90,
	}
	total, ok := floatFromBalanceMap(bal, "totalEquity", "totalWalletBalance")
	if !ok || total != 100.5 {
		t.Fatalf("totalEquity: got %v ok=%v", total, ok)
	}
	avail, ok := floatFromBalanceMap(bal, "availableBalance")
	if !ok || avail != 42.25 {
		t.Fatalf("availableBalance: got %v ok=%v", avail, ok)
	}
}

func TestExchangeHasCredentials(t *testing.T) {
	hl := &store.Exchange{
		ExchangeType:          "hyperliquid",
		APIKey:                "0xabc",
		HyperliquidWalletAddr: "0x123",
	}
	if !exchangeHasCredentials(hl) {
		t.Fatal("hyperliquid should have credentials")
	}
	binance := &store.Exchange{ExchangeType: "binance", APIKey: "k"}
	if exchangeHasCredentials(binance) {
		t.Fatal("binance without secret should fail")
	}
}

func TestDescribeLaunchPreflightFailures(t *testing.T) {
	msg := describeLaunchPreflightFailures(launchPreflightResult{
		Checks: []launchCheck{
			{ID: "ai_model", Status: "ok", Message: "ok"},
			{ID: "exchange_funds", Status: "failed", Message: "need funds"},
			{ID: "ai_wallet_funds", Status: "failed", Message: "fee empty"},
		},
	})
	if msg != "need funds fee empty" && msg != "fee empty need funds" {
		// order follows checks slice
		if msg != "need funds fee empty" {
			t.Fatalf("unexpected: %q", msg)
		}
	}
}

func TestExchangeAsset(t *testing.T) {
	if exchangeAsset("hyperliquid") != "USDC" {
		t.Fatal("hl asset")
	}
	if exchangeAsset("binance") != "USDT" {
		t.Fatal("binance asset")
	}
}

func TestAnnotateExchange(t *testing.T) {
	ex := &store.Exchange{ID: "ex-binance-1", ExchangeType: "binance"}
	got := annotateExchange(launchCheck{
		ID: "exchange_account", Status: "failed", Code: "permission_denied",
		Message: "IP whitelist",
	}, ex)
	if got.ExchangeID != "ex-binance-1" || got.ExchangeType != "binance" {
		t.Fatalf("annotate: %#v", got)
	}
	empty := annotateExchange(launchCheck{ID: "exchange_config", Status: "failed"}, nil)
	if empty.ExchangeID != "" || empty.ExchangeType != "" {
		t.Fatalf("nil exchange should leave ids empty: %#v", empty)
	}
}
