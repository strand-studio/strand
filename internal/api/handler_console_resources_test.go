package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"strand/internal/auth"
	"strand/internal/engine"
	"strand/internal/settings"
	"strand/internal/store"
)

func TestResources_DefaultsAndAgentProbe(t *testing.T) {
	auth.SetJWTSecret("test-secret-resources-0123456789")

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

	regBody := `{"email":"resources@example.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}
	var reg struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &reg); err != nil || reg.Token == "" {
		t.Fatalf("register token: %v body=%s", err, w.Body.String())
	}
	authHeader := "Bearer " + reg.Token

	// overview
	req = httptest.NewRequest(http.MethodGet, "/api/resources", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", w.Code, w.Body.String())
	}
	var overview map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if _, ok := overview["defaults"]; !ok {
		t.Fatalf("missing defaults: %#v", overview)
	}
	if _, ok := overview["agents"]; !ok {
		t.Fatalf("missing agents: %#v", overview)
	}
	if _, ok := overview["usage"]; !ok {
		t.Fatalf("missing usage: %#v", overview)
	}

	// save defaults
	put := `{"default_model_id":"m1","default_agent_id":"codex","default_exchange_id":"ex1","alert_platform_ids":["p1","p2"]}`
	req = httptest.NewRequest(http.MethodPut, "/api/resources/defaults", bytes.NewBufferString(put))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("defaults put status=%d body=%s", w.Code, w.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved["default_model_id"] != "m1" {
		t.Fatalf("default_model_id=%v", saved["default_model_id"])
	}
	ids, _ := saved["alert_platform_ids"].([]any)
	if len(ids) != 2 {
		t.Fatalf("alert ids=%v", saved["alert_platform_ids"])
	}

	// agent probe (may fail if binary missing — still records health)
	req = httptest.NewRequest(http.MethodPost, "/api/resources/agent/codex/test", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Fatalf("agent test status=%d body=%s", w.Code, w.Body.String())
	}
	var probe map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &probe); err != nil {
		t.Fatal(err)
	}
	if _, ok := probe["latency_ms"]; !ok {
		t.Fatalf("missing latency: %#v", probe)
	}

	// health persisted
	userID := "resources@example.com"
	// resolve actual user id from store users if needed
	h, err := st.Resources().GetHealth(userID, store.ResourceKindAgent, "codex")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	// If user id is uuid-based, try listing
	if h.LastTestAt == nil {
		rows, _ := st.Resources().ListHealth("", store.ResourceKindAgent)
		found := false
		for _, row := range rows {
			if row.ResourceID == "codex" && row.LastTestAt != nil {
				found = true
				h = row
				break
			}
		}
		if !found {
			// Register returns user_id — fetch via overview defaults user
			req = httptest.NewRequest(http.MethodGet, "/api/resources", nil)
			req.Header.Set("Authorization", authHeader)
			w = httptest.NewRecorder()
			r.ServeHTTP(w, req)
			// Soft-check: at least agent list includes health fields after test
			var ov2 map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &ov2)
			agents, _ := ov2["agents"].([]any)
			for _, a := range agents {
				m, _ := a.(map[string]any)
				if m["id"] == "codex" && m["tested"] == true {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected codex health recorded; probe=%#v", probe)
			}
		}
	}
	_ = time.Now()
}

func TestResources_UnknownKind(t *testing.T) {
	auth.SetJWTSecret("test-secret-resources-kind-0123")
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
	cfg.API.Token = "tok"
	cfg.API.Mode = "test"
	r := Handler(svc, cfg)

	regBody := `{"email":"kind@example.com","password":"secret12"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var reg struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &reg)

	req = httptest.NewRequest(http.MethodPost, "/api/resources/foo/bar/test", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer "+reg.Token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestResourceStore_RecordAndDefaults(t *testing.T) {
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "res.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	rs := st.Resources()
	if err := rs.RecordTest("u1", store.ResourceKindModel, "m1", true, 12, ""); err != nil {
		t.Fatal(err)
	}
	h, err := rs.GetHealth("u1", store.ResourceKindModel, "m1")
	if err != nil || !h.LastTestOK || h.LatencyMS != 12 {
		t.Fatalf("health=%#v err=%v", h, err)
	}

	d := &store.ResourceDefaults{UserID: "u1", DefaultModelID: "m1"}
	d.SetAlertIDs([]string{"a", "b"})
	if err := rs.SaveDefaults(d); err != nil {
		t.Fatal(err)
	}
	got, err := rs.GetDefaults("u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultModelID != "m1" || len(got.AlertIDs()) != 2 {
		t.Fatalf("defaults=%#v ids=%v", got, got.AlertIDs())
	}
}
