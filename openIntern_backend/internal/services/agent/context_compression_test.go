package agent

import (
	"context"
	"testing"

	"openIntern/internal/models"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

type stubModelCatalogResolver struct {
	selection *RuntimeModelSelection
}

type stubThreadContextSnapshotStore struct {
	latest  *models.ThreadContextSnapshot
	created []*models.ThreadContextSnapshot
}

// ResolveRuntimeSelection 返回预置模型选择结果，供压缩预算测试使用。
func (s stubModelCatalogResolver) ResolveRuntimeSelection(modelID, providerID string) (*RuntimeModelSelection, error) {
	return s.selection, nil
}

// GetLatestByThreadID returns the preconfigured latest snapshot.
func (s *stubThreadContextSnapshotStore) GetLatestByThreadID(threadID string) (*models.ThreadContextSnapshot, error) {
	return s.latest, nil
}

// Create records created snapshots for assertions.
func (s *stubThreadContextSnapshotStore) Create(item *models.ThreadContextSnapshot) error {
	s.created = append(s.created, item)
	s.latest = item
	return nil
}

// TestCompressInputContextNoTrigger verifies messages remain unchanged when soft limit is not exceeded.
func TestCompressInputContextNoTrigger(t *testing.T) {
	svc := &Service{}
	state := runtimeState{
		contextCompression: contextCompressionSettings{
			Enabled:                true,
			SoftLimitTokens:        200,
			HardLimitTokens:        300,
			MaxRecentMessages:      3,
			EstimatedCharsPerToken: 1,
		},
	}
	input := &types.RunAgentInput{
		Messages: []types.Message{
			{ID: "u-1", Role: types.RoleUser, Content: "hello"},
			{ID: "a-1", Role: types.RoleAssistant, Content: "world"},
		},
	}

	compressed, stats, err := svc.compressInputContext(context.Background(), input, nil, state)
	if err != nil {
		t.Fatalf("compressInputContext error: %v", err)
	}
	if compressed != input {
		t.Fatalf("expected original input pointer when no compression")
	}
	if stats == nil || stats.Triggered {
		t.Fatalf("expected compression not triggered, got %#v", stats)
	}
}

// TestCompressInputContextKeepPinnedAndRecent verifies system + last user are always preserved.
func TestCompressInputContextKeepPinnedAndRecent(t *testing.T) {
	svc := &Service{}
	state := runtimeState{
		contextCompression: contextCompressionSettings{
			Enabled:                true,
			SoftLimitTokens:        60,
			HardLimitTokens:        80,
			MaxRecentMessages:      2,
			EstimatedCharsPerToken: 1,
		},
	}
	input := &types.RunAgentInput{
		Messages: []types.Message{
			{ID: "s-1", Role: types.RoleSystem, Content: "SYS"},
			{ID: "u-1", Role: types.RoleUser, Content: "111111111111111111111111111111"},
			{ID: "a-1", Role: types.RoleAssistant, Content: "222222222222222222222222222222"},
			{ID: "u-2", Role: types.RoleUser, Content: "333333333333333333333333333333"},
			{ID: "a-2", Role: types.RoleAssistant, Content: "444444444444444444444444444444"},
			{ID: "u-3", Role: types.RoleUser, Content: "555555555555555555555555555555"},
		},
	}

	compressed, stats, err := svc.compressInputContext(context.Background(), input, nil, state)
	if err != nil {
		t.Fatalf("compressInputContext error: %v", err)
	}
	if stats == nil || !stats.Triggered {
		t.Fatalf("expected compression triggered, got %#v", stats)
	}
	if compressed == nil || len(compressed.Messages) != 2 {
		t.Fatalf("unexpected compressed messages: %#v", compressed)
	}
	if compressed.Messages[0].ID != "s-1" {
		t.Fatalf("expected system message kept, got %#v", compressed.Messages)
	}
	if compressed.Messages[1].ID != "u-3" {
		t.Fatalf("expected last user kept, got %#v", compressed.Messages)
	}
}

// TestCompressInputContextPinnedOverflow verifies request fails fast when pinned messages alone exceed hard limit.
func TestCompressInputContextPinnedOverflow(t *testing.T) {
	svc := &Service{}
	state := runtimeState{
		contextCompression: contextCompressionSettings{
			Enabled:                true,
			SoftLimitTokens:        100,
			HardLimitTokens:        150,
			MaxRecentMessages:      1,
			EstimatedCharsPerToken: 1,
		},
	}
	input := &types.RunAgentInput{
		Messages: []types.Message{
			{ID: "s-1", Role: types.RoleSystem, Content: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
			{ID: "u-1", Role: types.RoleUser, Content: "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"},
		},
	}

	_, _, err := svc.compressInputContext(context.Background(), input, nil, state)
	if err == nil {
		t.Fatal("expected error when pinned messages exceed hard limit")
	}
}

// TestBuildContextCompressionBudgetWithCapabilities verifies model capabilities can tighten hard limit.
func TestBuildContextCompressionBudgetWithCapabilities(t *testing.T) {
	svc := &Service{
		deps: Dependencies{
			ModelCatalogResolver: stubModelCatalogResolver{
				selection: &RuntimeModelSelection{
					Model: &models.ModelCatalog{
						CapabilitiesJSON: `{"context_window": 4096}`,
					},
				},
			},
		},
	}
	settings := contextCompressionSettings{
		Enabled:                true,
		SoftLimitTokens:        8000,
		HardLimitTokens:        9000,
		OutputReserveTokens:    1024,
		MaxRecentMessages:      5,
		EstimatedCharsPerToken: 4,
	}

	budget := svc.buildContextCompressionBudget(context.Background(), &AgentRuntimeConfig{
		Model: struct {
			ProviderID string
			ModelID    string
		}{
			ProviderID: "provider-1",
			ModelID:    "model-1",
		},
	}, settings)

	if budget.HardLimitTokens != 3072 {
		t.Fatalf("unexpected hard limit: %d", budget.HardLimitTokens)
	}
	if budget.SoftLimitTokens >= budget.HardLimitTokens {
		t.Fatalf("soft limit should be lower than hard limit, got soft=%d hard=%d", budget.SoftLimitTokens, budget.HardLimitTokens)
	}
}

// TestParseContextWindowTokens verifies multiple field styles are supported.
func TestParseContextWindowTokens(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "snake_case", raw: `{"context_window": 16384}`, want: 16384},
		{name: "camel_case", raw: `{"contextWindow": "32768"}`, want: 32768},
		{name: "nested_context", raw: `{"context":{"window":8192}}`, want: 8192},
		{name: "invalid", raw: `{"foo":"bar"}`, want: 0},
	}
	for _, testCase := range tests {
		got := parseContextWindowTokens(testCase.raw)
		if got != testCase.want {
			t.Fatalf("%s: got %d want %d", testCase.name, got, testCase.want)
		}
	}
}

// TestCompressInputContextWithSnapshotSummary verifies previous snapshot summary is updated and injected.
func TestCompressInputContextWithSnapshotSummary(t *testing.T) {
	store := &stubThreadContextSnapshotStore{
		latest: &models.ThreadContextSnapshot{
			ThreadID:          "thread-1",
			CompressionIndex:  3,
			CoveredUntilMsgID: "u-1",
			SummaryText:       "UserIntent: 老目标",
		},
	}
	svc := &Service{
		deps: Dependencies{
			ThreadContextSnapshotStore: store,
		},
	}
	state := runtimeState{
		contextCompression: contextCompressionSettings{
			Enabled:                true,
			SoftLimitTokens:        40,
			HardLimitTokens:        500,
			MaxRecentMessages:      0,
			EstimatedCharsPerToken: 1,
		},
	}
	input := &types.RunAgentInput{
		ThreadID: "thread-1",
		Messages: []types.Message{
			{ID: "s-1", Role: types.RoleSystem, Content: "SYS"},
			{ID: "u-1", Role: types.RoleUser, Content: "old-user"},
			{ID: "a-1", Role: types.RoleAssistant, Content: "old-assistant-content-xxxxxxxxxxxxxxxxxxxxxxxx"},
			{ID: "u-2", Role: types.RoleUser, Content: "latest-user-content-yyyyyyyyyyyyyyyyyyyyyyyyyyy"},
		},
	}

	compressed, stats, err := svc.compressInputContext(context.Background(), input, nil, state)
	if err != nil {
		t.Fatalf("compressInputContext error: %v", err)
	}
	if stats == nil || !stats.Triggered {
		t.Fatalf("expected compression triggered, got %#v", stats)
	}
	if !stats.SummaryUsed || !stats.SummaryUpdated {
		t.Fatalf("expected summary to be used and updated, got %#v", stats)
	}
	if stats.SnapshotIndex != 4 {
		t.Fatalf("unexpected snapshot index: %d", stats.SnapshotIndex)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected one new snapshot, got %d", len(store.created))
	}
	if store.created[0].CoveredUntilMsgID != "a-1" {
		t.Fatalf("unexpected covered_until_msg_id: %s", store.created[0].CoveredUntilMsgID)
	}
	if len(compressed.Messages) < 3 {
		t.Fatalf("expected summary message injected, got %#v", compressed.Messages)
	}
	if compressed.Messages[1].Role != types.RoleSystem {
		t.Fatalf("expected summary system message at index 1, got %#v", compressed.Messages)
	}
}
