package api

import (
	"encoding/json"
	"testing"
)

func TestNormalizeExchangeQuery(t *testing.T) {
	cases := map[string]string{
		"":                "hyperliquid",
		"hyperliquid-xyz": "hyperliquid",
		"HL":              "hyperliquid",
		"Binance":         "binance",
		"okx-swap":        "okx",
		"okex":            "okx",
	}
	for in, want := range cases {
		if got := normalizeExchangeQuery(in); got != want {
			t.Fatalf("normalizeExchangeQuery(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCoinCategory(t *testing.T) {
	if got := coinCategory("BTC"); got != "crypto" {
		t.Fatalf("BTC category=%q", got)
	}
	if got := coinCategory("xyz:TSLA"); got != "stock" {
		t.Fatalf("xyz:TSLA category=%q", got)
	}
	if got := coinCategory("xyz:GOLD"); got != "commodity" {
		t.Fatalf("xyz:GOLD category=%q", got)
	}
}

func TestChatToolDefsSchema(t *testing.T) {
	defs := chatToolDefs()
	if len(defs) < 6 {
		t.Fatalf("expected at least 6 tools, got %d", len(defs))
	}
	names := map[string]bool{}
	for _, d := range defs {
		if d.Type != "function" {
			t.Fatalf("tool type=%q", d.Type)
		}
		if d.Function.Name == "" {
			t.Fatal("empty tool name")
		}
		if d.Function.Parameters == nil {
			t.Fatalf("tool %s missing parameters", d.Function.Name)
		}
		names[d.Function.Name] = true
	}
	for _, required := range []string{
		"list_supported_chains",
		"list_markets",
		"get_klines",
		"get_direction_leaderboard",
		"market_brief",
	} {
		if !names[required] {
			t.Fatalf("missing tool %s", required)
		}
	}
}

func TestArgHelpers(t *testing.T) {
	args := map[string]any{
		"exchange": "binance",
		"limit":    float64(12),
		"empty":    "",
	}
	if got := argString(args, "exchange", "hyperliquid"); got != "binance" {
		t.Fatalf("argString exchange=%q", got)
	}
	if got := argString(args, "empty", "fallback"); got != "fallback" {
		t.Fatalf("argString empty=%q", got)
	}
	if got := argInt(args, "limit", 5); got != 12 {
		t.Fatalf("argInt limit=%d", got)
	}
	if got := argInt(args, "missing", 7); got != 7 {
		t.Fatalf("argInt missing=%d", got)
	}
}

func TestListSupportedChainsTool(t *testing.T) {
	svc := &Service{}
	raw := svc.executeChatTool(nil, "user", "list_supported_chains", `{}`)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	venues, ok := payload["venues"].([]any)
	if !ok || len(venues) < 5 {
		t.Fatalf("venues payload=%v", payload["venues"])
	}
}

func TestUnknownTool(t *testing.T) {
	svc := &Service{}
	raw := svc.executeChatTool(nil, "user", "does_not_exist", `{}`)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["error"] == nil {
		t.Fatalf("expected error, got %v", payload)
	}
}

func TestDecodeJSONObject(t *testing.T) {
	got := decodeJSONObject(`{"symbol":"BTC","limit":3}`)
	if got["symbol"] != "BTC" {
		t.Fatalf("symbol=%v", got["symbol"])
	}
	if got["limit"].(float64) != 3 {
		t.Fatalf("limit=%v", got["limit"])
	}
}

func TestToolResultJSON(t *testing.T) {
	raw := toolResultJSON(map[string]any{"ok": true})
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload=%v", payload)
	}
}
