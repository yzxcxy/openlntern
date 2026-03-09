package agent

import (
	"context"
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

// TestApplyForwardedPropsChainPluginSearch verifies plugin search config and query extraction.
func TestApplyForwardedPropsChainPluginSearch(t *testing.T) {
	input := &types.RunAgentInput{
		Messages: []types.Message{
			{ID: "u-1", Role: types.RoleUser, Content: "old question"},
			{
				ID:   "u-2",
				Role: types.RoleUser,
				Content: []any{
					map[string]any{"type": "text", "text": "latest question"},
					map[string]any{"type": "binary", "mimeType": "image/png", "url": "https://example.com/a.png"},
				},
			},
		},
		ForwardedProps: map[string]any{
			"agentConfig": map[string]any{
				"plugins": map[string]any{
					"mode": "search",
					"search": map[string]any{
						"topK":         12,
						"runtimeTypes": []any{" api ", "mcp", "invalid"},
						"minScore":     0.45,
						"maxMCPTools":  2,
					},
				},
			},
		},
	}

	runtimeConfig, err := applyForwardedPropsChain(context.Background(), input)
	if err != nil {
		t.Fatalf("applyForwardedPropsChain error: %v", err)
	}
	if runtimeConfig == nil {
		t.Fatal("runtimeConfig is nil")
	}
	if runtimeConfig.Plugins.Mode != "search" {
		t.Fatalf("unexpected plugin mode: %s", runtimeConfig.Plugins.Mode)
	}
	if runtimeConfig.Plugins.Search.TopK != 12 {
		t.Fatalf("unexpected topK: %d", runtimeConfig.Plugins.Search.TopK)
	}
	if runtimeConfig.Plugins.Search.MaxMCPTools != 2 {
		t.Fatalf("unexpected maxMCPTools: %d", runtimeConfig.Plugins.Search.MaxMCPTools)
	}
	if runtimeConfig.Plugins.Search.MinScore != 0.45 {
		t.Fatalf("unexpected minScore: %f", runtimeConfig.Plugins.Search.MinScore)
	}
	if len(runtimeConfig.Plugins.Search.RuntimeTypes) != 2 {
		t.Fatalf("unexpected runtime types: %#v", runtimeConfig.Plugins.Search.RuntimeTypes)
	}
	if runtimeConfig.Plugins.Search.RuntimeTypes[0] != "api" || runtimeConfig.Plugins.Search.RuntimeTypes[1] != "mcp" {
		t.Fatalf("unexpected runtime types order: %#v", runtimeConfig.Plugins.Search.RuntimeTypes)
	}
	if runtimeConfig.Plugins.SearchQuery != "latest question" {
		t.Fatalf("unexpected search query: %s", runtimeConfig.Plugins.SearchQuery)
	}
}

// TestFindLastUserMessageTextAndIndexOnlyText verifies non-text content does not fallback to string dump.
func TestFindLastUserMessageTextAndIndexOnlyText(t *testing.T) {
	messages := []types.Message{
		{ID: "u-1", Role: types.RoleUser, Content: "first"},
		{ID: "a-1", Role: types.RoleAssistant, Content: "answer"},
		{ID: "u-2", Role: types.RoleUser, Content: map[string]any{"foo": "bar"}},
	}

	text, index := findLastUserMessageTextAndIndex(messages)
	if index != 2 {
		t.Fatalf("unexpected user index: %d", index)
	}
	if text != "" {
		t.Fatalf("expected empty text, got: %s", text)
	}
}
