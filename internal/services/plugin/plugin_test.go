package plugin

import (
	"testing"

	"openIntern/internal/models"
)

func TestDiffPluginTools(t *testing.T) {
	existing := []models.Tool{
		{ToolID: "tool-1", ToolName: "existing"},
		{ToolID: "tool-2", ToolName: "removed"},
	}
	incoming := []models.Tool{
		{ToolID: "tool-1", ToolName: "updated"},
		{ToolID: "tool-3", ToolName: "created"},
	}

	toUpdate, toCreate, removed := diffPluginTools(existing, incoming)

	if len(toUpdate) != 1 || toUpdate[0].ToolID != "tool-1" {
		t.Fatalf("unexpected toolsToUpdate: %#v", toUpdate)
	}
	if len(toCreate) != 1 || toCreate[0].ToolID != "tool-3" {
		t.Fatalf("unexpected toolsToCreate: %#v", toCreate)
	}
	if len(removed) != 1 || removed[0] != "tool-2" {
		t.Fatalf("unexpected removed tool ids: %#v", removed)
	}
}

func TestBuildToolCodeLanguageValidation(t *testing.T) {
	svc := &PluginService{}

	tool, err := svc.buildTool("plugin-1", pluginRuntimeCode, PluginToolInput{
		ToolName:     "run_python",
		Code:         "print('ok')",
		CodeLanguage: " PYTHON ",
	})
	if err != nil {
		t.Fatalf("buildTool returned error: %v", err)
	}
	if tool.CodeLanguage != codeLanguagePython {
		t.Fatalf("unexpected code language: %s", tool.CodeLanguage)
	}
	if tool.ToolResponseMode != toolResponseNonStreaming {
		t.Fatalf("unexpected tool response mode: %s", tool.ToolResponseMode)
	}

	_, err = svc.buildTool("plugin-1", pluginRuntimeCode, PluginToolInput{
		ToolName:     "run_ruby",
		Code:         "puts 'ok'",
		CodeLanguage: "ruby",
	})
	if err == nil {
		t.Fatal("expected invalid code language error")
	}
}

func TestBuildToolRejectsInvalidToolName(t *testing.T) {
	svc := &PluginService{}

	_, err := svc.buildTool("plugin-1", pluginRuntimeCode, PluginToolInput{
		ToolName:     "echo input",
		Code:         "print('ok')",
		CodeLanguage: "python",
	})
	if err == nil {
		t.Fatal("expected invalid tool name error")
	}
	if got := err.Error(); got != "tool_name must match ^[a-zA-Z0-9_-]+$" {
		t.Fatalf("unexpected error: %s", got)
	}
}
