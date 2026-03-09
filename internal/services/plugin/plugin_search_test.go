package plugin

import "testing"

// TestNormalizeToolSearchRuntimeTypes verifies runtime type filtering and defaults.
func TestNormalizeToolSearchRuntimeTypes(t *testing.T) {
	got := normalizeToolSearchRuntimeTypes([]string{" API ", "mcp", "invalid", "code", "mcp"})
	if len(got) != 3 {
		t.Fatalf("unexpected runtime types length: %#v", got)
	}
	if got[0] != pluginRuntimeAPI || got[1] != pluginRuntimeMCP || got[2] != pluginRuntimeCode {
		t.Fatalf("unexpected runtime types: %#v", got)
	}

	defaulted := normalizeToolSearchRuntimeTypes([]string{"invalid"})
	if len(defaulted) != 3 {
		t.Fatalf("expected default runtime types, got: %#v", defaulted)
	}
}

// TestNormalizeToolSearchBounds verifies fixed topK and maxMCP normalization fallback.
func TestNormalizeToolSearchBounds(t *testing.T) {
	if normalizeToolSearchTopK(0) != defaultToolSearchTopK {
		t.Fatalf("unexpected default topK")
	}
	if normalizeToolSearchTopK(5) != defaultToolSearchTopK {
		t.Fatalf("topK should stay fixed")
	}
	if normalizeToolSearchMaxMCPTools(-1) != defaultToolSearchMaxMCPTools {
		t.Fatalf("unexpected default maxMCPTools")
	}
	if normalizeToolSearchMaxMCPTools(2) != 2 {
		t.Fatalf("unexpected normalized maxMCPTools")
	}
}

// TestNormalizeToolSearchScoreThreshold verifies score threshold fallback and bounds.
func TestNormalizeToolSearchScoreThreshold(t *testing.T) {
	if normalizeToolSearchScoreThreshold(0) != defaultToolSearchScoreThreshold {
		t.Fatalf("unexpected default score threshold")
	}
	if normalizeToolSearchScoreThreshold(0.6) != 0.6 {
		t.Fatalf("unexpected score threshold")
	}
	if normalizeToolSearchScoreThreshold(1.5) != 1 {
		t.Fatalf("score threshold should be capped at 1")
	}
}
