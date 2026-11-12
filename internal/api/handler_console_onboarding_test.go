package api

import (
	"testing"
	"time"
)

func TestCachedBaseUSDCBalanceCachesUnknown(t *testing.T) {
	addr := "0x00000000000000000000000000000000000000aa"
	beginnerBalanceCache.Store(addr, balanceCacheEntry{
		balance: "1.250000",
		status:  "ok",
		at:      time.Now(),
	})
	bal, status := cachedBaseUSDCBalance(addr)
	if bal != "1.250000" || status != "ok" {
		t.Fatalf("got bal=%q status=%q", bal, status)
	}
}

func TestBeginnerDefaultModel(t *testing.T) {
	if beginnerDefaultModel != "deepseek" {
		t.Fatalf("expected deepseek beginner default, got %q", beginnerDefaultModel)
	}
}
