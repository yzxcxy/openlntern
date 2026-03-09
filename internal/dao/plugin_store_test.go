package dao

import (
	"os"
	"path/filepath"
	"testing"

	"openIntern/internal/config"
	"openIntern/internal/database"
)

// TestToolStoreURIBuilders verifies root/plugin/resource URI normalization.
func TestToolStoreURIBuilders(t *testing.T) {
	database.InitContextStore(config.OpenVikingConfig{
		BaseURL:   "http://127.0.0.1:1933",
		ToolsRoot: "viking://resources/tools",
	})

	root := Plugin.ToolStoreRootURI()
	if root != "viking://resources/tools/" {
		t.Fatalf("unexpected root uri: %s", root)
	}

	pluginURI := Plugin.ToolStorePluginURI("plugin-1")
	if pluginURI != "viking://resources/tools/plugin-1/" {
		t.Fatalf("unexpected plugin uri: %s", pluginURI)
	}

	resourceURI := Plugin.ToolStoreResourceURI("plugin-1", "tool-1")
	if resourceURI != "viking://resources/tools/plugin-1/tool-1.md" {
		t.Fatalf("unexpected resource uri: %s", resourceURI)
	}
}

// TestNormalizeToolStoreEntryURI verifies fs entries normalization.
func TestNormalizeToolStoreEntryURI(t *testing.T) {
	pluginURI := "viking://resources/tools/plugin-1/"
	cases := []struct {
		name      string
		entryPath string
		entryName string
		expected  string
	}{
		{
			name:      "full_uri",
			entryPath: "viking://resources/tools/plugin-1/tool-a.md",
			expected:  "viking://resources/tools/plugin-1/tool-a.md",
		},
		{
			name:      "relative_path",
			entryPath: "tool-b.md",
			expected:  "viking://resources/tools/plugin-1/tool-b.md",
		},
		{
			name:      "name_only",
			entryName: "tool-c.md",
			expected:  "viking://resources/tools/plugin-1/tool-c.md",
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := normalizeToolStoreEntryURI(pluginURI, item.entryPath, item.entryName)
			if got != item.expected {
				t.Fatalf("unexpected uri: got=%s want=%s", got, item.expected)
			}
		})
	}
}

// TestWriteToolStoreDocumentFiles verifies batch file staging behavior for plugin tool documents.
func TestWriteToolStoreDocumentFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tool-store-docs-")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	docs := map[string]string{
		"tool-a": "doc-a",
		"tool-b": "doc-b",
	}
	if err := writeToolStoreDocumentFiles(tempDir, docs); err != nil {
		t.Fatalf("write tool docs failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, "tool-a.md"))
	if err != nil {
		t.Fatalf("read staged file failed: %v", err)
	}
	if string(content) != "doc-a" {
		t.Fatalf("unexpected staged content: %s", string(content))
	}
}

// TestWriteToolStoreDocumentFilesInvalidToolID verifies invalid tool_id validation during batch staging.
func TestWriteToolStoreDocumentFilesInvalidToolID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tool-store-docs-invalid-")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	docs := map[string]string{
		"bad/tool": "doc",
	}
	if err := writeToolStoreDocumentFiles(tempDir, docs); err == nil {
		t.Fatalf("expected invalid tool_id error")
	}
}
