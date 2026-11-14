package core

import "testing"

func TestLooksLikeAgentAuthRequired(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Not logged in · Please run /login", true},
		{"Please run /login", true},
		{"Claude Code is not logged in; run `claude auth login`", true},
		{"未登录，请先登录 Claude Code", true},
		{"Here is how to structure a login form in React…", false},
		{"", false},
		{"Hello, how can I help?", false},
	}
	for _, tc := range cases {
		if got := looksLikeAgentAuthRequired(tc.in); got != tc.want {
			t.Fatalf("looksLikeAgentAuthRequired(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}
