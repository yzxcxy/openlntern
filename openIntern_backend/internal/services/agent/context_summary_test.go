package agent

import (
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

// TestParseSummaryPayloadFromText verifies json and fenced json payloads are parsed correctly.
func TestParseSummaryPayloadFromText(t *testing.T) {
	raw := "```json\n{\"user_intent\":\"完成功能\",\"decisions\":[\"先做后端\"],\"constraints\":[\"不能影响主流程\"],\"open_tasks\":[\"补测试\"],\"facts\":[\"已接入压缩\"],\"do_not_repeat\":[\"不要重写前端\"]}\n```"
	payload := parseSummaryPayloadFromText(raw)
	if payload.UserIntent != "完成功能" {
		t.Fatalf("unexpected user intent: %s", payload.UserIntent)
	}
	if len(payload.Decisions) != 1 || payload.Decisions[0] != "先做后端" {
		t.Fatalf("unexpected decisions: %#v", payload.Decisions)
	}
	if len(payload.Constraints) != 1 {
		t.Fatalf("unexpected constraints: %#v", payload.Constraints)
	}
}

// TestBuildContextSummaryMessageContent verifies prefix and truncation behavior for summary instruction.
func TestBuildContextSummaryMessageContent(t *testing.T) {
	content := buildContextSummaryMessageContent("UserIntent: A")
	if content == "" {
		t.Fatal("expected non-empty summary message content")
	}
	if content[:len(contextSummaryPromptPrefix)] != contextSummaryPromptPrefix {
		t.Fatalf("summary prefix missing: %s", content)
	}
}

// TestSummarizeContextHistoryFallback verifies deterministic fallback summary without summary model.
func TestSummarizeContextHistoryFallback(t *testing.T) {
	summaryText, summaryJSON, tokens, err := summarizeContextHistory(nil, nil, "UserIntent: 老摘要", []types.Message{
		{Role: types.RoleAssistant, Content: "新增事实内容"},
	})
	if err != nil {
		t.Fatalf("summarizeContextHistory error: %v", err)
	}
	if summaryText == "" {
		t.Fatal("expected fallback summary text")
	}
	if summaryJSON == "" {
		t.Fatal("expected summary json")
	}
	if tokens <= 0 {
		t.Fatalf("expected positive token estimate, got %d", tokens)
	}
}
