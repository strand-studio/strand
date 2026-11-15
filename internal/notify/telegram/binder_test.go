package telegram

import (
	"strings"
	"testing"
)

type stubStatusProvider struct{ text string }

func (s stubStatusProvider) TelegramStatusText() string { return s.text }

func TestStatusTextPrefersProvider(t *testing.T) {
	b := NewBinder(nil)
	b.SetStatusProvider(stubStatusProvider{
		text: "Strand is online.\n1 Autopilot(s) running:\n• Bot · ⏸ daily loss",
	})
	got := b.statusText()
	if !strings.Contains(got, "⏸ daily loss") {
		t.Fatalf("got %q", got)
	}
}

func TestStatusTextFallbackNoProvider(t *testing.T) {
	b := NewBinder(nil)
	got := b.statusText()
	if got != "Store unavailable" {
		t.Fatalf("got %q", got)
	}
}

func TestFirstNonEmptyTG(t *testing.T) {
	if firstNonEmptyTG("", "hi") != "hi" {
		t.Fatal("expected hi")
	}
}

func TestHelpCommandParsing(t *testing.T) {
	cases := []string{"/help", "/help@MyBot", "/START", "/start@bot"}
	for _, text := range cases {
		fields := strings.Fields(text)
		cmd, _, _ := strings.Cut(fields[0], "@")
		cmd = strings.ToLower(cmd)
		if cmd != "/help" && cmd != "/start" {
			t.Fatalf("got %q from %q", cmd, text)
		}
	}
}
