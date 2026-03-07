package agent

import (
	"strings"
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

// TestNormalizeForwardedContextSelection verifies forwarded contextSelections are decoded correctly.
func TestNormalizeForwardedContextSelection(t *testing.T) {
	raw := map[string]any{
		"skills": []any{
			map[string]any{"id": "skill-1", "name": "Skill One"},
		},
		"knowledgeBases": []any{
			map[string]any{"id": "kb-a", "name": "Knowledge A"},
		},
	}
	selection, err := normalizeForwardedContextSelection(raw)
	if err != nil {
		t.Fatalf("normalizeForwardedContextSelection error: %v", err)
	}
	if selection == nil {
		t.Fatalf("normalizeForwardedContextSelection got nil")
	}
	if len(selection.Skills) != 1 || selection.Skills[0].ID != "skill-1" {
		t.Fatalf("unexpected skills: %#v", selection.Skills)
	}
	if len(selection.KnowledgeBases) != 1 || selection.KnowledgeBases[0].ID != "kb-a" {
		t.Fatalf("unexpected knowledge bases: %#v", selection.KnowledgeBases)
	}
}

// TestInjectMessageBeforeUserAt verifies temporary instructions are inserted before the target user message.
func TestInjectMessageBeforeUserAt(t *testing.T) {
	input := &types.RunAgentInput{
		Messages: []types.Message{
			{ID: "u-1", Role: types.RoleUser, Content: "question"},
		},
	}
	injectMessageBeforeUserAt(input, 0, types.Message{
		ID:      "sys-1",
		Role:    types.RoleSystem,
		Content: "context",
	})
	if len(input.Messages) != 2 {
		t.Fatalf("unexpected messages length: %d", len(input.Messages))
	}
	if input.Messages[0].ID != "sys-1" || input.Messages[1].ID != "u-1" {
		t.Fatalf("unexpected message order: %#v", input.Messages)
	}
}

// TestBuildSkillInstructionMessage verifies selected skills are rendered into instruction text.
func TestBuildSkillInstructionMessage(t *testing.T) {
	message := buildSkillInstructionMessage([]forwardedContextTarget{
		{ID: "skill-a", Name: "Skill A"},
		{ID: "skill-b", Name: "Skill B"},
	})
	if message == "" {
		t.Fatalf("buildSkillInstructionMessage got empty message")
	}
	if !strings.Contains(message, "Skill A") || !strings.Contains(message, "Skill B") {
		t.Fatalf("unexpected instruction message: %s", message)
	}
}
