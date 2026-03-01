package services

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
