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
	if normalizeToolSearchTopK(5) != 5 {
		t.Fatalf("topK should preserve explicit values within bounds")
	}
	if normalizeToolSearchMaxMCPTools(-1) != defaultToolSearchMaxMCPTools {
		t.Fatalf("unexpected default maxMCPTools")
	}
	if normalizeToolSearchMaxMCPTools(2) != 2 {
		t.Fatalf("unexpected normalized maxMCPTools")
	}
}

// TestParseToolSearchQuery verifies select and required-term parsing.
func TestParseToolSearchQuery(t *testing.T) {
	selectQuery, ok := parseToolSearchQuery("select:tool_a, tool_b")
	if !ok || !selectQuery.IsSelect {
		t.Fatalf("expected select query to parse successfully")
	}
	if len(selectQuery.SelectedNames) != 2 {
		t.Fatalf("unexpected selected names: %#v", selectQuery.SelectedNames)
	}

	keywordQuery, ok := parseToolSearchQuery("+slack send")
	if !ok || keywordQuery.IsSelect {
		t.Fatalf("expected keyword query to parse successfully")
	}
	if len(keywordQuery.RequiredTerms) != 1 || keywordQuery.RequiredTerms[0] != "slack" {
		t.Fatalf("unexpected required terms: %#v", keywordQuery.RequiredTerms)
	}
	if len(keywordQuery.OptionalTerms) != 1 || keywordQuery.OptionalTerms[0] != "send" {
		t.Fatalf("unexpected optional terms: %#v", keywordQuery.OptionalTerms)
	}
}
