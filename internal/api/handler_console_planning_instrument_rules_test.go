package api

import "testing"

func TestTrimDecimalZeros(t *testing.T) {
	cases := map[string]string{
		"0.01000000": "0.01",
		"1.000":      "1",
		"5":          "5",
		"":           "",
	}
	for in, want := range cases {
		if got := trimDecimalZeros(in); got != want {
			t.Fatalf("%q → %q want %q", in, got, want)
		}
	}
}
