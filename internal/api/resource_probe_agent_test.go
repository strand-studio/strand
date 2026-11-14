package api

import "testing"

func TestParseClaudeAuthLoggedIn(t *testing.T) {
	ok, err := parseClaudeAuthLoggedIn(`{"loggedIn":true,"authMethod":"oauth"}`)
	if err != nil || !ok {
		t.Fatalf("logged in json: ok=%v err=%v", ok, err)
	}
	ok, err = parseClaudeAuthLoggedIn(`{"loggedIn":false,"authMethod":"none"}`)
	if err != nil || ok {
		t.Fatalf("logged out json: ok=%v err=%v", ok, err)
	}
	ok, err = parseClaudeAuthLoggedIn("noise\n{\"loggedIn\": true}\n")
	if err != nil || !ok {
		t.Fatalf("wrapped json: ok=%v err=%v", ok, err)
	}
	if _, err := parseClaudeAuthLoggedIn("not json"); err == nil {
		t.Fatal("expected parse error")
	}
}
