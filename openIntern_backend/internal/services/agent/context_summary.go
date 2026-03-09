package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"openIntern/internal/models"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/schema"
)

const (
	contextSummaryPromptPrefix     = "以下是历史对话压缩摘要，仅用于保持上下文连续性，请在回答时优先遵守其中的约束和已确认结论："
	contextSummaryMessageMaxChars  = 2400
	contextSummarySourceMaxChars   = 6000
	contextSummaryPreviousMaxChars = 2000
)

// contextSummaryPayload defines the structured summary schema used for storage and prompt replay.
type contextSummaryPayload struct {
	UserIntent  string   `json:"user_intent"`
	Decisions   []string `json:"decisions"`
	Constraints []string `json:"constraints"`
	OpenTasks   []string `json:"open_tasks"`
	Facts       []string `json:"facts"`
	DoNotRepeat []string `json:"do_not_repeat"`
}

// buildContextSummaryMessageContent renders summary text into a stable system instruction block.
func buildContextSummaryMessageContent(summaryText string) string {
	trimmed := strings.TrimSpace(summaryText)
	if trimmed == "" {
		return ""
	}
	if len([]rune(trimmed)) > contextSummaryMessageMaxChars {
		trimmed = string([]rune(trimmed)[:contextSummaryMessageMaxChars])
	}
	return contextSummaryPromptPrefix + "\n" + trimmed
}

// summarizeContextHistory generates a structured compression summary from previous summary and newly removed messages.
func summarizeContextHistory(ctx context.Context, summaryModel *deepseek.ChatModel, previousSummary string, removedMessages []types.Message) (string, string, int, error) {
	previous := strings.TrimSpace(previousSummary)
	source := buildSummarySourceFromMessages(removedMessages)
	if previous == "" && source == "" {
		return "", "", 0, nil
	}
	if summaryModel == nil {
		fallback := buildFallbackSummary(previous, source)
		payload := contextSummaryPayload{
			UserIntent: firstNonEmptyLine(fallback),
			Facts:      []string{fallback},
		}
		return fallback, marshalSummaryPayload(payload), estimateSummaryTokens(fallback), nil
	}

	userPrompt := buildSummaryUserPrompt(previous, source)
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是对话上下文压缩器。请输出 JSON 对象，不要包含 markdown 代码块。字段必须包含：user_intent(string)、decisions(string[])、constraints(string[])、open_tasks(string[])、facts(string[])、do_not_repeat(string[])。每个数组最多 6 条，每条不超过 40 字。",
		},
		{
			Role:    schema.User,
			Content: userPrompt,
		},
	}
	resp, err := summaryModel.Generate(ctx, messages)
	if err != nil {
		return "", "", 0, err
	}
	payload := parseSummaryPayloadFromText(resp.Content)
	summaryText := renderSummaryText(payload)
	if summaryText == "" {
		summaryText = buildFallbackSummary(previous, source)
	}
	return summaryText, marshalSummaryPayload(payload), estimateSummaryTokens(summaryText), nil
}

// buildSummaryUserPrompt composes model input with previous summary and newly removed message digest.
func buildSummaryUserPrompt(previousSummary string, source string) string {
	parts := []string{
		"请基于以下历史信息生成新的压缩摘要。",
	}
	previous := strings.TrimSpace(previousSummary)
	if previous != "" {
		parts = append(parts, "【已有压缩摘要】\n"+truncateByRunes(previous, contextSummaryPreviousMaxChars))
	}
	if strings.TrimSpace(source) != "" {
		parts = append(parts, "【本轮新增被压缩的历史消息】\n"+source)
	}
	parts = append(parts, "要求：保留已确认约束和决策，不要臆造事实。")
	return strings.Join(parts, "\n\n")
}

// buildSummarySourceFromMessages converts removed messages into compact role-text lines for summary generation.
func buildSummarySourceFromMessages(messages []types.Message) string {
	if len(messages) == 0 {
		return ""
	}
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		content := strings.TrimSpace(stringifyMessageContent(message.Content))
		if content == "" {
			continue
		}
		content = strings.ReplaceAll(content, "\n", " ")
		content = strings.Join(strings.Fields(content), " ")
		if content == "" {
			continue
		}
		if len([]rune(content)) > 240 {
			content = string([]rune(content)[:240])
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", message.Role, content))
	}
	joined := strings.Join(lines, "\n")
	return truncateByRunes(joined, contextSummarySourceMaxChars)
}

// parseSummaryPayloadFromText extracts structured payload from model output and applies defensive normalization.
func parseSummaryPayloadFromText(raw string) contextSummaryPayload {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return contextSummaryPayload{}
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		trimmed = trimmed[start : end+1]
	}
	var payload contextSummaryPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return contextSummaryPayload{
			UserIntent: firstNonEmptyLine(raw),
			Facts:      []string{truncateByRunes(strings.TrimSpace(raw), 240)},
		}
	}
	payload.UserIntent = truncateByRunes(strings.TrimSpace(payload.UserIntent), 120)
	payload.Decisions = sanitizeSummaryItems(payload.Decisions, 6)
	payload.Constraints = sanitizeSummaryItems(payload.Constraints, 6)
	payload.OpenTasks = sanitizeSummaryItems(payload.OpenTasks, 6)
	payload.Facts = sanitizeSummaryItems(payload.Facts, 6)
	payload.DoNotRepeat = sanitizeSummaryItems(payload.DoNotRepeat, 6)
	return payload
}

// sanitizeSummaryItems normalizes summary arrays by trimming blanks and enforcing max item count.
func sanitizeSummaryItems(items []string, maxCount int) []string {
	if maxCount <= 0 {
		return []string{}
	}
	sanitized := make([]string, 0, len(items))
	for _, item := range items {
		value := truncateByRunes(strings.TrimSpace(item), 80)
		if value == "" {
			continue
		}
		sanitized = append(sanitized, value)
		if len(sanitized) >= maxCount {
			break
		}
	}
	return sanitized
}

// renderSummaryText renders structured payload into deterministic markdown-like sections.
func renderSummaryText(payload contextSummaryPayload) string {
	lines := make([]string, 0, 32)
	if payload.UserIntent != "" {
		lines = append(lines, "UserIntent: "+payload.UserIntent)
	}
	appendSection := func(title string, values []string) {
		if len(values) == 0 {
			return
		}
		lines = append(lines, "")
		lines = append(lines, title+":")
		for _, value := range values {
			lines = append(lines, "- "+value)
		}
	}
	appendSection("Decisions", payload.Decisions)
	appendSection("Constraints", payload.Constraints)
	appendSection("OpenTasks", payload.OpenTasks)
	appendSection("Facts", payload.Facts)
	appendSection("DoNotRepeat", payload.DoNotRepeat)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// marshalSummaryPayload serializes structured summary payload for database storage.
func marshalSummaryPayload(payload contextSummaryPayload) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// estimateSummaryTokens estimates summary token count using the same coarse char/token ratio.
func estimateSummaryTokens(summaryText string) int {
	trimmed := strings.TrimSpace(summaryText)
	if trimmed == "" {
		return 0
	}
	chars := len(trimmed)
	return (chars + defaultEstimatedCharsPerToken - 1) / defaultEstimatedCharsPerToken
}

// buildFallbackSummary provides a deterministic fallback when model output is unavailable.
func buildFallbackSummary(previousSummary string, source string) string {
	previous := strings.TrimSpace(previousSummary)
	incremental := strings.TrimSpace(source)
	switch {
	case previous != "" && incremental != "":
		return truncateByRunes(previous+"\n\n增量事实:\n- "+firstNonEmptyLine(incremental), contextSummaryMessageMaxChars)
	case previous != "":
		return truncateByRunes(previous, contextSummaryMessageMaxChars)
	default:
		return truncateByRunes("UserIntent: "+firstNonEmptyLine(incremental), contextSummaryMessageMaxChars)
	}
}

// firstNonEmptyLine returns the first non-empty line from text.
func firstNonEmptyLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return truncateByRunes(trimmed, 120)
		}
	}
	return ""
}

// truncateByRunes truncates UTF-8 text by rune length and keeps original text when short enough.
func truncateByRunes(raw string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(raw)
	if len(runes) <= maxRunes {
		return raw
	}
	return string(runes[:maxRunes])
}

// buildAndPersistThreadSummary merges removed messages into snapshot summary and persists a new snapshot record.
func (s *Service) buildAndPersistThreadSummary(ctx context.Context, threadID string, removedMessages []types.Message, previousSnapshot *models.ThreadContextSnapshot, summaryModel *deepseek.ChatModel) (*models.ThreadContextSnapshot, string, error) {
	if strings.TrimSpace(threadID) == "" {
		return nil, "", nil
	}
	if len(removedMessages) == 0 && previousSnapshot == nil {
		return nil, "", nil
	}
	previousSummary := ""
	if previousSnapshot != nil {
		previousSummary = previousSnapshot.SummaryText
	}
	summaryText, summaryStructJSON, approxTokens, err := summarizeContextHistory(ctx, summaryModel, previousSummary, removedMessages)
	if err != nil {
		return nil, "", err
	}
	summaryText = strings.TrimSpace(summaryText)
	if summaryText == "" {
		return nil, "", nil
	}
	if previousSnapshot != nil && len(removedMessages) == 0 {
		return previousSnapshot, summaryText, nil
	}

	nextIndex := 1
	if previousSnapshot != nil && previousSnapshot.CompressionIndex >= 1 {
		nextIndex = previousSnapshot.CompressionIndex + 1
	}
	coveredUntil := ""
	if len(removedMessages) > 0 {
		coveredUntil = strings.TrimSpace(removedMessages[len(removedMessages)-1].ID)
	}
	item := &models.ThreadContextSnapshot{
		ThreadID:          strings.TrimSpace(threadID),
		CompressionIndex:  nextIndex,
		CoveredUntilMsgID: coveredUntil,
		SummaryText:       summaryText,
		SummaryStructJSON: strings.TrimSpace(summaryStructJSON),
		ApproxTokens:      approxTokens,
	}
	if s.deps.ThreadContextSnapshotStore != nil {
		if err := s.deps.ThreadContextSnapshotStore.Create(item); err != nil {
			return nil, "", err
		}
	}
	return item, summaryText, nil
}
