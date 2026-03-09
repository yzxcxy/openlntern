package dao

import "testing"

// TestDeriveMemoryTypeFromURI verifies OpenViking memory categories are inferred from their URIs.
func TestDeriveMemoryTypeFromURI(t *testing.T) {
	cases := []struct {
		name string
		uri  string
		want string
	}{
		{name: "profile overview", uri: "viking://user/memories/.overview.md", want: "profile"},
		{name: "profile file", uri: "viking://user/memories/profile.md", want: "profile"},
		{name: "preferences", uri: "viking://user/memories/preferences/coding/style.md", want: "preferences"},
		{name: "entities", uri: "viking://user/memories/entities/project/openintern.md", want: "entities"},
		{name: "events", uri: "viking://user/memories/events/decision-1.md", want: "events"},
		{name: "cases", uri: "viking://agent/memories/cases/oauth-login.md", want: "cases"},
		{name: "patterns", uri: "viking://agent/memories/patterns/debug-flow.md", want: "patterns"},
		{name: "unknown", uri: "viking://resources/docs/auth.md", want: ""},
	}

	for _, item := range cases {
		if got := deriveMemoryTypeFromURI(item.uri); got != item.want {
			t.Fatalf("%s: expected %q, got %q", item.name, item.want, got)
		}
	}
}

// TestPickMemoryContextsFallsBackToResources verifies older payloads still produce retrievable matches.
func TestPickMemoryContextsFallsBackToResources(t *testing.T) {
	result := openVikingMemoryFindResult{
		Resources: []openVikingMatchedContext{{URI: "viking://user/memories/preferences/coding.md"}},
	}

	got := pickMemoryContexts(result)
	if len(got) != 1 {
		t.Fatalf("expected one fallback item, got %d", len(got))
	}
	if got[0].URI != "viking://user/memories/preferences/coding.md" {
		t.Fatalf("unexpected fallback uri: %s", got[0].URI)
	}
}

// TestIsMemoryURIUnderTargetAcceptsTenantUserMemory verifies tenant-scoped user memory URIs still match the logical user memory root.
func TestIsMemoryURIUnderTargetAcceptsTenantUserMemory(t *testing.T) {
	candidate := "viking://user/default/memories/preferences/style.md"
	target := "viking://user/memories/"
	if !isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s to match target %s", candidate, target)
	}
}

// TestIsMemoryURIUnderTargetAcceptsTenantAgentMemory verifies tenant-scoped agent memory URIs still match the logical agent memory root.
func TestIsMemoryURIUnderTargetAcceptsTenantAgentMemory(t *testing.T) {
	candidate := "viking://agent/default/memories/patterns/debug.md"
	target := "viking://agent/memories/"
	if !isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s to match target %s", candidate, target)
	}
}
