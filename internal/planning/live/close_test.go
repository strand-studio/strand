package live

import "testing"

func TestSamePlanningSymbol(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"BTCUSDT", "BTCUSDT", true},
		{"BTC-USDT-SWAP", "BTCUSDT", true},
		{"xyz:TSLA", "XYZ:TSLA", true},
		{"ETHUSDT", "BTCUSDT", false},
		{"", "BTCUSDT", false},
	}
	for _, tc := range cases {
		if got := SamePlanningSymbol(tc.a, tc.b); got != tc.want {
			t.Fatalf("SamePlanningSymbol(%q,%q)=%v want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestMatchCloseQuantityAndEntry(t *testing.T) {
	if !MatchCloseQuantity(0.01, 0.01) {
		t.Fatal("exact quantity should match")
	}
	if MatchCloseQuantity(0.01, 0.009) {
		t.Fatal("smaller venue size must not match")
	}
	if MatchCloseQuantity(0.01, 0.02) {
		t.Fatal("larger venue size must not match")
	}
	if !MatchCloseEntry(100.5, 100.5) {
		t.Fatal("exact entry should match")
	}
	if MatchCloseEntry(100.5, 101) {
		t.Fatal("material entry change must not match")
	}
}
