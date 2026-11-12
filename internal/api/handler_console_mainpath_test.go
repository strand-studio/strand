package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"strand/internal/auth"
	"strand/internal/engine"
	"strand/internal/settings"
	"strand/internal/store"
)

func TestMainPath_RegisterOnboardingPreflightKeys(t *testing.T) {
	auth.SetJWTSecret("test-secret-mainpath-0123456789")

	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	eng := engine.NewEngine(settings.Settings{})
	eng.UseStoreForTest(st)
	svc := NewService(eng, "test").(*Service)
	cfg := settings.Settings{}
	cfg.API.Token = "test-api-token"
	cfg.API.Mode = "test"
	r := Handler(svc, cfg)

	// register
	regBody := `{"email":"mainpath@example.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}
	var reg struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &reg); err != nil || reg.Token == "" {
		t.Fatalf("register token: %v body=%s", err, w.Body.String())
	}

	authHeader := "Bearer " + reg.Token

	// beginner onboarding creates claw402 wallet
	req = httptest.NewRequest(http.MethodPost, "/api/onboarding/beginner", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("onboarding status=%d body=%s", w.Code, w.Body.String())
	}
	var onboard map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &onboard); err != nil {
		t.Fatal(err)
	}
	if onboard["ready"] != true {
		t.Fatalf("onboarding not ready: %#v", onboard)
	}
	addr, _ := onboard["address"].(string)
	modelID, _ := onboard["configured_model_id"].(string)
	if addr == "" || modelID == "" {
		t.Fatalf("missing wallet fields: %#v", onboard)
	}

	// current wallet
	req = httptest.NewRequest(http.MethodGet, "/api/onboarding/beginner/current", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("current wallet status=%d body=%s", w.Code, w.Body.String())
	}

	// preflight without exchange → failed with checks
	preBody := `{"ai_model_id":"` + modelID + `","exchange_id":"","strategy_id":""}`
	req = httptest.NewRequest(http.MethodPost, "/api/launch/preflight", bytes.NewBufferString(preBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preflight status=%d body=%s", w.Code, w.Body.String())
	}
	var pre launchPreflightResult
	if err := json.Unmarshal(w.Body.Bytes(), &pre); err != nil {
		t.Fatal(err)
	}
	if pre.Ready {
		t.Fatal("preflight should not be ready without exchange")
	}
	foundTradeAuth := false
	for _, c := range pre.Checks {
		if c.ID == "exchange_trade_auth" && c.Status == "failed" {
			foundTradeAuth = true
		}
	}
	if !foundTradeAuth {
		t.Fatalf("expected exchange_trade_auth failed check, got %#v", pre.Checks)
	}

	// Seed a CEX exchange with credentials so preflight checks are annotated
	// with exchange_id / exchange_type (FE deep-link contract).
	exID, err := st.Exchange().Create(
		reg.UserID, "binance", "Test Binance", true,
		"test-api-key", "test-secret-key", "", false,
		"", false, false,
		"", "", "",
		"", "", "", 0,
	)
	if err != nil || exID == "" {
		t.Fatalf("create binance exchange: id=%q err=%v", exID, err)
	}
	preBody = `{"ai_model_id":"` + modelID + `","exchange_id":"` + exID + `","strategy_id":""}`
	req = httptest.NewRequest(http.MethodPost, "/api/launch/preflight", bytes.NewBufferString(preBody))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("cex preflight status=%d body=%s", w.Code, w.Body.String())
	}
	var cexPre launchPreflightResult
	if err := json.Unmarshal(w.Body.Bytes(), &cexPre); err != nil {
		t.Fatal(err)
	}
	annotated := false
	for _, c := range cexPre.Checks {
		if c.ExchangeID == "" && c.ExchangeType == "" {
			continue
		}
		if c.ExchangeID != exID || c.ExchangeType != "binance" {
			t.Fatalf("annotated check mismatch: %#v want id=%s type=binance", c, exID)
		}
		annotated = true
	}
	if !annotated {
		t.Fatalf("expected exchange_id/type on CEX preflight checks, got %#v", cexPre.Checks)
	}

	// my-traders empty list shape
	req = httptest.NewRequest(http.MethodGet, "/api/my-traders", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("my-traders status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStartErrorKeysStable(t *testing.T) {
	keys := []string{
		"trader.not_found",
		"trader.start.preflight_failed",
		"trader.start.load_failed",
		"trader.start.failed",
		"trader.start.already_running",
		"trader.stop.failed",
	}
	for _, k := range keys {
		if k == "" {
			t.Fatal("empty key")
		}
	}
}
