package tools

import (
	"testing"
)

func TestIsModelSafeToolName(t *testing.T) {
	cases := map[string]bool{
		"echo_input": true,
		"echo-input": true,
		"echo input": false,
		"中文":         false,
		"":           false,
	}

	for name, want := range cases {
		if got := IsModelSafeToolName(name); got != want {
			t.Fatalf("unexpected validation result for %q: got %v want %v", name, got, want)
		}
	}
}
