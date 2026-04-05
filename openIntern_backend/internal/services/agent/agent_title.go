package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ensureThreadTitle 在线程无标题时基于最近用户输入生成并写回标题。
func ensureThreadTitle(ctx context.Context, threadStore ThreadStore, userID, threadID string, messages []types.Message, titleModel einoModel.ToolCallingChatModel) error {
	if strings.TrimSpace(userID) == "" {
		return nil
	}
	if threadID == "" {
		return nil
	}
	thread, err := threadStore.GetThreadByThreadID(userID, threadID)
	if err != nil || thread == nil {
		return err
	}
	if strings.TrimSpace(thread.Title) != "" {
		return nil
	}
	source := extractTitleSource(messages)
	if source == "" {
		return nil
	}
	title, err := generateTitle(ctx, source, titleModel)
	if err != nil {
		return err
	}
	title = sanitizeTitle(title)
	if title == "" {
		return nil
	}
	return threadStore.UpdateThreadTitle(userID, thread.ThreadID, title)
}

// extractTitleSource 提取可用于生成标题的最近一条用户文本内容。
func extractTitleSource(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == types.RoleUser {
			switch v := msg.Content.(type) {
			case string:
				return strings.TrimSpace(v)
			case nil:
				return ""
			default:
				return strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return ""
}

// generateTitle 调用标题模型生成简短标题文本。
func generateTitle(ctx context.Context, content string, titleModel einoModel.ToolCallingChatModel) (string, error) {
	if titleModel == nil {
		return "", nil
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}
	if len(trimmed) > 600 {
		trimmed = trimmed[:600]
	}
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "请为用户对话生成简短中文标题，不超过20字，只输出标题本身。",
		},
		{
			Role:    schema.User,
			Content: trimmed,
		},
	}
	resp, err := titleModel.Generate(ctx, messages)
	if err != nil || resp == nil {
		return "", err
	}
	return resp.Content, nil
}

// sanitizeTitle 清洗并限制标题长度，避免持久化异常内容。
func sanitizeTitle(title string) string {
	cleaned := strings.TrimSpace(title)
	cleaned = strings.Trim(cleaned, "“”\"'")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	runes := []rune(cleaned)
	if len(runes) > 40 {
		return string(runes[:40])
	}
	return cleaned
}
