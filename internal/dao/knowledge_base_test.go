package dao

import (
	"testing"

	"openIntern/internal/config"
	"openIntern/internal/database"
)

// TestKnowledgeBaseReservedTopLevelDirs verifies tools root is excluded from KB namespace.
func TestKnowledgeBaseReservedTopLevelDirs(t *testing.T) {
	database.InitContextStore(config.OpenVikingConfig{
		BaseURL:   "http://127.0.0.1:1933",
		ToolsRoot: "viking://resources/tools",
	})

	dao := &KnowledgeBaseDAO{}
	reserved := dao.reservedTopLevelDirs()
	if _, ok := reserved["tools"]; !ok {
		t.Fatalf("expected tools dir to be reserved: %#v", reserved)
	}
	if !dao.isReservedTopLevelDir("tools") {
		t.Fatalf("expected tools to be reserved")
	}
	if dao.isReservedTopLevelDir("project-docs") {
		t.Fatalf("unexpected reserved kb name")
	}
}
