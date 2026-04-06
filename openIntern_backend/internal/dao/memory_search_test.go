package dao

import "testing"

// TestDeriveMemoryTypeFromURI verifies OpenViking memory categories are inferred from their URIs.
func TestDeriveMemoryTypeFromURI(t *testing.T) {
	cases := []struct {
		name string
		uri  string
		want string
	}{
		{name: "profile overview", uri: "viking://user/user-123/memories/.overview.md", want: "profile"},
		{name: "profile file", uri: "viking://user/user-123/memories/profile.md", want: "profile"},
		{name: "preferences", uri: "viking://user/user-123/memories/preferences/coding/style.md", want: "preferences"},
		{name: "entities", uri: "viking://user/user-123/memories/entities/project/openintern.md", want: "entities"},
		{name: "events", uri: "viking://user/user-123/memories/events/decision-1.md", want: "events"},
		{name: "cases", uri: "viking://agent/default/memories/cases/oauth-login.md", want: "cases"},
		{name: "patterns", uri: "viking://agent/default/memories/patterns/debug-flow.md", want: "patterns"},
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
		Resources: []openVikingMatchedContext{{URI: "viking://user/user-123/memories/preferences/coding.md"}},
	}

	got := pickMemoryContexts(result)
	if len(got) != 1 {
		t.Fatalf("expected one fallback item, got %d", len(got))
	}
	if got[0].URI != "viking://user/user-123/memories/preferences/coding.md" {
		t.Fatalf("unexpected fallback uri: %s", got[0].URI)
	}
}

// TestIsMemoryURIUnderTargetAcceptsUserMemory verifies user memory URIs match the same user root.
func TestIsMemoryURIUnderTargetAcceptsUserMemory(t *testing.T) {
	candidate := "viking://user/user-123/memories/preferences/style.md"
	target := "viking://user/user-123/memories/"
	if !isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s to match target %s", candidate, target)
	}
}

// TestIsMemoryURIUnderTargetRejectsLegacyUserMemoryRoot verifies legacy roots are no longer accepted.
func TestIsMemoryURIUnderTargetRejectsLegacyUserMemoryRoot(t *testing.T) {
	candidate := "viking://user/user-123/memories/preferences/style.md"
	target := "viking://user/memories/"
	if isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s not to match target %s", candidate, target)
	}
}

// TestIsMemoryURIUnderTargetRejectsDifferentUserSpace verifies one user root does not match another user space.
func TestIsMemoryURIUnderTargetRejectsDifferentUserSpace(t *testing.T) {
	candidate := "viking://user/another-space/memories/preferences/style.md"
	target := "viking://user/user-123/memories/"
	if isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s not to match target %s", candidate, target)
	}
}

// TestIsMemoryURIUnderTargetAcceptsDefaultAgentMemory verifies default agent memory URIs match the default root.
func TestIsMemoryURIUnderTargetAcceptsDefaultAgentMemory(t *testing.T) {
	candidate := "viking://agent/default/memories/patterns/debug.md"
	target := "viking://agent/default/memories/"
	if !isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s to match target %s", candidate, target)
	}
}

// TestIsMemoryURIUnderTargetRejectsLegacyAgentMemoryRoot verifies legacy agent roots are no longer accepted.
func TestIsMemoryURIUnderTargetRejectsLegacyAgentMemoryRoot(t *testing.T) {
	candidate := "viking://agent/default/memories/patterns/debug.md"
	target := "viking://agent/memories/"
	if isMemoryURIUnderTarget(candidate, target) {
		t.Fatalf("expected candidate %s not to match target %s", candidate, target)
	}
}
