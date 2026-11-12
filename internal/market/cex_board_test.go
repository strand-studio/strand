package market

import (
	"context"
	"testing"
	"time"
)

func TestFetchCEXBoardSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skip live CEX board in short mode")
	}
	exchanges := []string{"binance", "bybit", "okx", "gate", "bitget"}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	for _, ex := range exchanges {
		ex := ex
		t.Run(ex, func(t *testing.T) {
			rows, err := FetchCEXBoard(ctx, ex, 50)
			if err != nil {
				t.Skipf("FetchCEXBoard(%s) unavailable in this network: %v", ex, err)
			}
			if len(rows) == 0 {
				t.Fatalf("%s: empty board", ex)
			}
			priced := 0
			for _, row := range rows {
				if row.MarkPrice > 0 {
					priced++
				}
				if row.Symbol == "" || row.Display == "" {
					t.Fatalf("%s: missing symbol/display: %+v", ex, row)
				}
				if row.Exchange != ex {
					t.Fatalf("%s: exchange mismatch %q", ex, row.Exchange)
				}
			}
			if priced == 0 {
				t.Fatalf("%s: no priced rows", ex)
			}
			t.Logf("%s: %d rows, top=%s price=%.4f chg=%.2f%% vol=%.0f",
				ex, len(rows), rows[0].Display, rows[0].MarkPrice, rows[0].Change24hPct, rows[0].Volume24h)
		})
	}
}

func TestSupportsLiveCEXBoard(t *testing.T) {
	if !SupportsLiveCEXBoard("binance") {
		t.Fatal("expected binance support")
	}
	if SupportsLiveCEXBoard("lighter") {
		t.Fatal("lighter should not claim live board yet")
	}
}
