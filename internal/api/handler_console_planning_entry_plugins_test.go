package api

import (
	"fmt"
	"strconv"
	"testing"

	"strand/internal/planning/core"
)

func TestRunBuiltinEntryPluginRiskSize(t *testing.T) {
	out, err := runBuiltinEntryPlugin("risk-size-v1", entryPluginInput{
		Equity:   1000,
		RiskPct:  1,
		Entry:    100,
		Stop:     95,
		Leverage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	qty, _ := strconv.ParseFloat(stringifyPluginValue(out["quantity"]), 64)
	// 1% of 1000 = $10 risk / $5 stop = 2 contracts; notional 200 < 10k leverage cap
	if qty < 1.9 || qty > 2.1 {
		t.Fatalf("quantity=%v want ~2", qty)
	}
	if stringifyPluginValue(out["capped_by"]) != "risk" {
		t.Fatalf("capped_by=%v", out["capped_by"])
	}
}

func TestRunBuiltinEntryPluginLeverageCap(t *testing.T) {
	out, err := runBuiltinEntryPlugin("risk-size-v1", entryPluginInput{
		Equity:   100,
		RiskPct:  50,
		Entry:    100,
		Stop:     99,
		Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stringifyPluginValue(out["capped_by"]) != "leverage" {
		t.Fatalf("capped_by=%v want leverage", out["capped_by"])
	}
}

func TestRunBuiltinEntryPluginNotionalCap(t *testing.T) {
	out, err := runBuiltinEntryPlugin("notional-cap-v1", entryPluginInput{
		AvailableMargin: 50,
		Leverage:        2,
		Entry:           100,
	})
	if err != nil {
		t.Fatal(err)
	}
	qty, _ := strconv.ParseFloat(stringifyPluginValue(out["quantity"]), 64)
	if qty < 0.99 || qty > 1.01 {
		t.Fatalf("quantity=%v want ~1", qty)
	}
}

func TestBuiltinEntryPluginsListed(t *testing.T) {
	plugins := builtinEntryPlugins("sess-1")
	if len(plugins) < 2 {
		t.Fatalf("len=%d", len(plugins))
	}
	if stringifyPluginValue(plugins[0]["session_id"]) != "sess-1" {
		t.Fatalf("session_id=%v", plugins[0]["session_id"])
	}
}

func stringifyPluginValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case core.Decimal:
		return string(t)
	default:
		return fmt.Sprint(v)
	}
}
