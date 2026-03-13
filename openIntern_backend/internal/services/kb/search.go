package kb

import (
	"context"
	"fmt"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/util"
)

const contextPrefix = "以下是当前问题在所选知识库中的检索结果，请优先参考这些信息后再回答："

func (s *Service) BuildContextMessage(ctx context.Context, query string, kbNames []string, limitPerKB int) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return "", nil
	}
	normalizedNames := normalizeKnowledgeBaseNames(kbNames)
	if len(normalizedNames) == 0 {
		return "", nil
	}
	matches, err := dao.KnowledgeBase.SearchInKnowledgeBases(ctx, query, normalizedNames, limitPerKB)
	if err != nil {
		return "", err
	}
	return formatContextMessage(matches), nil
}

func normalizeKnowledgeBaseNames(names []string) []string {
	result := make([]string, 0, len(names))
	for _, name := range names {
		kbName, err := normalizeKnowledgeBaseName(name)
		if err != nil {
			continue
		}
		result = append(result, kbName)
	}
	return util.NormalizeUniqueStringList(result)
}

func formatContextMessage(matches []dao.KnowledgeBaseSearchMatch) string {
	if len(matches) == 0 {
		return ""
	}
	lines := make([]string, 0, len(matches)*3+1)
	lines = append(lines, contextPrefix)
	for i, item := range matches {
		kbName := strings.TrimSpace(item.KnowledgeBaseName)
		if kbName == "" {
			kbName = "未命名知识库"
		}
		abstract := strings.TrimSpace(item.Abstract)
		if abstract == "" {
			abstract = "(无摘要)"
		}
		lines = append(lines, fmt.Sprintf("%d. [知识库:%s]", i+1, kbName))
		lines = append(lines, "uri: "+strings.TrimSpace(item.URI))
		lines = append(lines, "摘要: "+abstract)
	}
	return strings.Join(lines, "\n")
}
