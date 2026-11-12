package fanout

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Fatalf("got %q", got)
	}
	got := truncate(strings.Repeat("a", 20), 10)
	if !strings.HasSuffix(got, "…") || len([]rune(got)) != 11 {
		t.Fatalf("got %q", got)
	}
}

func TestDiscordWebhookPayload(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	a := &Alerter{client: srv.Client()}
	row := &struct {
		WebhookURL string
	}{WebhookURL: srv.URL}
	// exercise postJSON path used by Discord webhooks
	if err := a.postJSON(row.WebhookURL, map[string]any{"content": "hi"}); err != nil {
		t.Fatal(err)
	}
	if got["content"] != "hi" {
		t.Fatalf("payload %#v", got)
	}
}

func TestSlackWebhookPayload(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	a := &Alerter{client: srv.Client()}
	if err := a.postJSON(srv.URL, map[string]any{"text": "alert"}); err != nil {
		t.Fatal(err)
	}
	if got["text"] != "alert" {
		t.Fatalf("payload %#v", got)
	}
}
