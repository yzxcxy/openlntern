package dao

import "testing"

// TestExtractToolIDFromVikingURI verifies tool_id parsing from viking resource URI.
func TestExtractToolIDFromVikingURI(t *testing.T) {
	cases := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "markdown file",
			uri:  "viking://resources/tools/tenant/tool-123.md",
			want: "tool-123",
		},
		{
			name: "plain segment",
			uri:  "viking://resources/tools/tenant/tool-xyz",
			want: "tool-xyz",
		},
		{
			name: "query and fragment",
			uri:  "viking://resources/tools/tenant/tool-a.md?x=1#frag",
			want: "tool-a",
		},
		{
			name: "overview metadata file",
			uri:  "viking://resources/tools/tenant/tool-b/.overview.md",
			want: "tool-b",
		},
		{
			name: "abstract metadata file",
			uri:  "viking://resources/tools/tenant/tool-c/.abstract.md?x=1",
			want: "tool-c",
		},
		{
			name: "empty",
			uri:  "",
			want: "",
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := extractToolIDFromVikingURI(item.uri)
			if got != item.want {
				t.Fatalf("extractToolIDFromVikingURI(%q)=%q, want=%q", item.uri, got, item.want)
			}
		})
	}
}

// TestIsToolSearchURIUnderTarget verifies URI prefix filtering for tool search results.
func TestIsToolSearchURIUnderTarget(t *testing.T) {
	cases := []struct {
		name      string
		uri       string
		targetURI string
		want      bool
	}{
		{
			name:      "within target",
			uri:       "viking://resources/tools/plugin-a/tool-1/tool-1.md",
			targetURI: "viking://resources/tools/plugin-a/",
			want:      true,
		},
		{
			name:      "outside target",
			uri:       "viking://resources/tools/plugin-b/tool-2/tool-2.md",
			targetURI: "viking://resources/tools/plugin-a/",
			want:      false,
		},
		{
			name:      "empty uri",
			uri:       "",
			targetURI: "viking://resources/tools/plugin-a/",
			want:      false,
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := isToolSearchURIUnderTarget(item.uri, item.targetURI)
			if got != item.want {
				t.Fatalf("isToolSearchURIUnderTarget(%q, %q)=%v, want=%v", item.uri, item.targetURI, got, item.want)
			}
		})
	}
}

// TestExtractToolIDFromVikingURIWithTarget verifies tool_id extraction with target-aware path validation.
func TestExtractToolIDFromVikingURIWithTarget(t *testing.T) {
	cases := []struct {
		name      string
		uri       string
		targetURI string
		want      string
	}{
		{
			name:      "tool content file",
			uri:       "viking://resources/tools/plugin-a/tool-1/tool-1.md",
			targetURI: "viking://resources/tools/",
			want:      "tool-1",
		},
		{
			name:      "tool metadata file",
			uri:       "viking://resources/tools/plugin-a/tool-2/.overview.md",
			targetURI: "viking://resources/tools/",
			want:      "tool-2",
		},
		{
			name:      "plugin metadata file should be ignored",
			uri:       "viking://resources/tools/plugin-a/.overview.md",
			targetURI: "viking://resources/tools/",
			want:      "",
		},
		{
			name:      "outside target",
			uri:       "viking://resources/golang/doc.md",
			targetURI: "viking://resources/tools/",
			want:      "",
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := extractToolIDFromVikingURIWithTarget(item.uri, item.targetURI)
			if got != item.want {
				t.Fatalf("extractToolIDFromVikingURIWithTarget(%q, %q)=%q, want=%q", item.uri, item.targetURI, got, item.want)
			}
		})
	}
}
