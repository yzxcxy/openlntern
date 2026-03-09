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
