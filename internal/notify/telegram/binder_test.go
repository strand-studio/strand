package telegram

import (
	"strings"
	"testing"
)

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
