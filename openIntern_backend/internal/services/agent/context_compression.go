package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"openIntern/internal/config"
	"openIntern/internal/models"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

const (
	defaultCompressionHardLimitTokens     = 12000
	defaultCompressionOutputReserveTokens = 2048
	defaultCompressionMaxRecentMessages   = 14
	defaultEstimatedCharsPerToken         = 4
)

// contextCompressionSettings 定义运行时上下文压缩参数。
type contextCompressionSettings struct {
	Enabled                bool
	SoftLimitTokens        int
	HardLimitTokens        int
	OutputReserveTokens    int
	MaxRecentMessages      int
	EstimatedCharsPerToken int
}

// contextCompressionBudget 定义单轮请求可使用的输入预算。
type contextCompressionBudget struct {
	SoftLimitTokens     int
	HardLimitTokens     int
	ContextWindowTokens int
	OutputReserveTokens int
}

// contextCompressionStats 记录压缩执行结果，用于日志与观测指标。
type contextCompressionStats struct {
	Enabled          bool
	Triggered        bool
	OriginalTokens   int
	CompressedTokens int
	RemovedMessages  int
	SoftLimitTokens  int
	HardLimitTokens  int
	SummaryUsed      bool
	SummaryUpdated   bool
	SnapshotIndex    int
}

// newContextCompressionSettings 根据配置构建压缩参数，并填充默认值。
func newContextCompressionSettings(cfg config.ContextCompressionConfig) contextCompressionSettings {
	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}

	outputReserve := cfg.OutputReserveTokens
	if outputReserve <= 0 {
		outputReserve = defaultCompressionOutputReserveTokens
	}
	maxRecent := cfg.MaxRecentMessages
	if maxRecent <= 0 {
		maxRecent = defaultCompressionMaxRecentMessages
	}
	charsPerToken := cfg.EstimatedCharsPerToken
	if charsPerToken <= 0 {
		charsPerToken = defaultEstimatedCharsPerToken
	}

	softLimit := cfg.SoftLimitTokens
	hardLimit := cfg.HardLimitTokens
	if hardLimit <= 0 {
		hardLimit = defaultCompressionHardLimitTokens
	}
	if softLimit <= 0 || softLimit >= hardLimit {
		softLimit = hardLimit * 85 / 100
		if softLimit <= 0 {
			softLimit = hardLimit
		}
	}

	return contextCompressionSettings{
		Enabled:                enabled,
		SoftLimitTokens:        softLimit,
		HardLimitTokens:        hardLimit,
		OutputReserveTokens:    outputReserve,
		MaxRecentMessages:      maxRecent,
		EstimatedCharsPerToken: charsPerToken,
	}
}

// compressInputContext 在模型调用前执行预算感知压缩，优先保留关键消息与最近窗口。
func (s *Service) compressInputContext(ctx context.Context, input *types.RunAgentInput, runtimeConfig *AgentRuntimeConfig, state runtimeState) (*types.RunAgentInput, *contextCompressionStats, error) {
	if input == nil {
		return nil, nil, nil
	}

	settings := state.contextCompression
	budget := s.buildContextCompressionBudget(ctx, runtimeConfig, settings)
	stats := &contextCompressionStats{
		Enabled:         settings.Enabled,
		SoftLimitTokens: budget.SoftLimitTokens,
		HardLimitTokens: budget.HardLimitTokens,
	}
	if !settings.Enabled || len(input.Messages) == 0 {
		return input, stats, nil
	}

	messageTokens := estimateMessageTokens(input.Messages, settings.EstimatedCharsPerToken)
	originalTokens := sumIntSlice(messageTokens)
	stats.OriginalTokens = originalTokens
	if originalTokens <= budget.SoftLimitTokens {
		stats.CompressedTokens = originalTokens
		return input, stats, nil
	}
	stats.Triggered = true
	latestSnapshot := s.loadLatestThreadContextSnapshot(input.ThreadID)
	coveredIndex := -1
	if latestSnapshot != nil && strings.TrimSpace(latestSnapshot.CoveredUntilMsgID) != "" {
		coveredIndex = findMessageIndexByID(input.Messages, latestSnapshot.CoveredUntilMsgID)
	}

	n := len(input.Messages)
	pinned := make([]bool, n)
	for i, msg := range input.Messages {
		if msg.Role == types.RoleSystem {
			pinned[i] = true
		}
	}
	if lastUserIndex := findLastUserMessageIndex(input.Messages); lastUserIndex >= 0 {
		pinned[lastUserIndex] = true
	}

	kept := make([]bool, n)
	for i := range kept {
		kept[i] = pinned[i]
	}
	recentCount := 0
	for i := n - 1; i >= 0 && recentCount < settings.MaxRecentMessages; i-- {
		if coveredIndex >= 0 && i <= coveredIndex {
			continue
		}
		if kept[i] {
			continue
		}
		kept[i] = true
		recentCount++
	}

	keptTokens := sumSelectedTokens(messageTokens, kept)
	for i := 0; i < n && keptTokens > budget.HardLimitTokens; i++ {
		if !kept[i] || pinned[i] {
			continue
		}
		kept[i] = false
		keptTokens -= messageTokens[i]
	}
	if keptTokens > budget.HardLimitTokens {
		return nil, stats, fmt.Errorf("context exceeds hard limit after compression: estimated_tokens=%d hard_limit=%d", keptTokens, budget.HardLimitTokens)
	}

	compressedMessages := make([]types.Message, 0, n)
	removedMessagesForSummary := make([]types.Message, 0, n)
	for i, msg := range input.Messages {
		if kept[i] {
			compressedMessages = append(compressedMessages, msg)
			continue
		}
		if i > coveredIndex {
			removedMessagesForSummary = append(removedMessagesForSummary, msg)
		}
	}

	summaryText := ""
	if latestSnapshot != nil {
		summaryText = latestSnapshot.SummaryText
		stats.SnapshotIndex = latestSnapshot.CompressionIndex
	}
	if len(removedMessagesForSummary) > 0 || summaryText != "" {
		snapshot, mergedSummary, err := s.buildAndPersistThreadSummary(ctx, input.ThreadID, removedMessagesForSummary, latestSnapshot, state.titleModel)
		if err == nil {
			if strings.TrimSpace(mergedSummary) != "" {
				summaryText = mergedSummary
			}
			if snapshot != nil {
				stats.SnapshotIndex = snapshot.CompressionIndex
				if len(removedMessagesForSummary) > 0 {
					stats.SummaryUpdated = true
				}
			}
		}
	}
	summaryContent := buildContextSummaryMessageContent(summaryText)
	if summaryContent != "" {
		summaryMessage := types.Message{
			ID:      events.GenerateMessageID(),
			Role:    types.RoleSystem,
			Content: summaryContent,
		}
		summaryTokens := estimateMessageTokens([]types.Message{summaryMessage}, settings.EstimatedCharsPerToken)
		summaryCost := sumIntSlice(summaryTokens)
		if summaryCost > 0 && keptTokens+summaryCost <= budget.HardLimitTokens {
			compressedMessages = prependSystemSummaryMessage(compressedMessages, summaryMessage)
			keptTokens += summaryCost
			stats.SummaryUsed = true
		}
	}

	stats.CompressedTokens = keptTokens
	removedCount := 0
	for _, keep := range kept {
		if !keep {
			removedCount++
		}
	}
	stats.RemovedMessages = removedCount
	if stats.RemovedMessages <= 0 && !stats.SummaryUsed {
		return input, stats, nil
	}
	return cloneRunAgentInputWithMessages(input, compressedMessages), stats, nil
}

// buildContextCompressionBudget 结合基础配置和模型能力，计算本轮有效上下文预算。
func (s *Service) buildContextCompressionBudget(ctx context.Context, runtimeConfig *AgentRuntimeConfig, settings contextCompressionSettings) contextCompressionBudget {
	budget := contextCompressionBudget{
		SoftLimitTokens:     settings.SoftLimitTokens,
		HardLimitTokens:     settings.HardLimitTokens,
		OutputReserveTokens: settings.OutputReserveTokens,
	}

	modelWindow := s.resolveRuntimeContextWindow(ctx, runtimeConfig)
	if modelWindow > 0 {
		budget.ContextWindowTokens = modelWindow
		modelHardLimit := modelWindow - settings.OutputReserveTokens
		if modelHardLimit < 1 {
			modelHardLimit = 1
		}
		if budget.HardLimitTokens <= 0 || modelHardLimit < budget.HardLimitTokens {
			budget.HardLimitTokens = modelHardLimit
		}
	}
	if budget.HardLimitTokens <= 0 {
		budget.HardLimitTokens = defaultCompressionHardLimitTokens
	}
	if budget.SoftLimitTokens <= 0 || budget.SoftLimitTokens >= budget.HardLimitTokens {
		budget.SoftLimitTokens = budget.HardLimitTokens * 85 / 100
		if budget.SoftLimitTokens <= 0 {
			budget.SoftLimitTokens = budget.HardLimitTokens
		}
	}
	return budget
}

// resolveRuntimeContextWindow 解析运行时模型能力中的 context window 大小。
func (s *Service) resolveRuntimeContextWindow(ctx context.Context, runtimeConfig *AgentRuntimeConfig) int {
	if s == nil || s.deps.ModelCatalogResolver == nil || runtimeConfig == nil {
		return 0
	}
	modelID := strings.TrimSpace(runtimeConfig.Model.ModelID)
	providerID := strings.TrimSpace(runtimeConfig.Model.ProviderID)
	if modelID == "" && providerID == "" {
		return 0
	}
	selection, err := s.deps.ModelCatalogResolver.ResolveRuntimeSelection(modelID, providerID)
	if err != nil || selection == nil || selection.Model == nil {
		return 0
	}
	return parseContextWindowTokens(selection.Model.CapabilitiesJSON)
}

// parseContextWindowTokens 从 capabilities_json 中提取上下文窗口大小。
func parseContextWindowTokens(capabilitiesJSON string) int {
	trimmed := strings.TrimSpace(capabilitiesJSON)
	if trimmed == "" {
		return 0
	}
	var capabilities map[string]any
	if err := json.Unmarshal([]byte(trimmed), &capabilities); err != nil {
		return 0
	}
	for _, key := range []string{"context_window", "contextWindow", "max_input_tokens", "maxInputTokens"} {
		if value, ok := capabilities[key]; ok {
			if tokens := parsePositiveInt(value); tokens > 0 {
				return tokens
			}
		}
	}
	if contextValue, ok := capabilities["context"].(map[string]any); ok {
		for _, key := range []string{"window", "context_window", "max_input_tokens"} {
			if value, ok := contextValue[key]; ok {
				if tokens := parsePositiveInt(value); tokens > 0 {
					return tokens
				}
			}
		}
	}
	return 0
}

// parsePositiveInt 将任意 JSON 值转换为正整数，失败返回 0。
func parsePositiveInt(value any) int {
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 && v <= int64(^uint(0)>>1) {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	case json.Number:
		if i, err := v.Int64(); err == nil && i > 0 && i <= int64(^uint(0)>>1) {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i > 0 {
			return i
		}
	}
	return 0
}

// estimateMessageTokens 按字符近似估算消息 token。
func estimateMessageTokens(messages []types.Message, charsPerToken int) []int {
	if charsPerToken <= 0 {
		charsPerToken = defaultEstimatedCharsPerToken
	}
	result := make([]int, 0, len(messages))
	for _, msg := range messages {
		payload := strings.TrimSpace(string(msg.Role) + " " + stringifyMessageContent(msg.Content))
		if payload == "" {
			result = append(result, 1)
			continue
		}
		chars := len(payload)
		tokens := (chars + charsPerToken - 1) / charsPerToken
		if tokens < 1 {
			tokens = 1
		}
		result = append(result, tokens)
	}
	return result
}

// stringifyMessageContent 将任意消息内容归一化为字符串，便于估算 token。
func stringifyMessageContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// findLastUserMessageIndex 返回最后一条 user 消息索引，未找到时返回 -1。
func findLastUserMessageIndex(messages []types.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == types.RoleUser {
			return i
		}
	}
	return -1
}

// findMessageIndexByID returns the first matched message index by id.
func findMessageIndexByID(messages []types.Message, msgID string) int {
	target := strings.TrimSpace(msgID)
	if target == "" {
		return -1
	}
	for index, message := range messages {
		if strings.TrimSpace(message.ID) == target {
			return index
		}
	}
	return -1
}

// prependSystemSummaryMessage inserts summary right after existing system messages.
func prependSystemSummaryMessage(messages []types.Message, summaryMessage types.Message) []types.Message {
	insertAt := 0
	for insertAt < len(messages) && messages[insertAt].Role == types.RoleSystem {
		insertAt++
	}
	next := make([]types.Message, 0, len(messages)+1)
	next = append(next, messages[:insertAt]...)
	next = append(next, summaryMessage)
	next = append(next, messages[insertAt:]...)
	return next
}

// loadLatestThreadContextSnapshot reads latest snapshot and hides store errors from request path.
func (s *Service) loadLatestThreadContextSnapshot(threadID string) *models.ThreadContextSnapshot {
	if s == nil || s.deps.ThreadContextSnapshotStore == nil {
		return nil
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil
	}
	item, err := s.deps.ThreadContextSnapshotStore.GetLatestByThreadID(threadID)
	if err != nil {
		return nil
	}
	return item
}

// sumIntSlice 计算整数切片总和。
func sumIntSlice(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

// sumSelectedTokens 计算被保留消息的估算 token 总和。
func sumSelectedTokens(tokens []int, selected []bool) int {
	total := 0
	for i := range tokens {
		if i < len(selected) && selected[i] {
			total += tokens[i]
		}
	}
	return total
}

// cloneRunAgentInputWithMessages 复制 RunAgentInput 并替换消息列表。
func cloneRunAgentInputWithMessages(input *types.RunAgentInput, messages []types.Message) *types.RunAgentInput {
	if input == nil {
		return nil
	}
	cloned := *input
	cloned.Messages = append([]types.Message(nil), messages...)
	return &cloned
}
