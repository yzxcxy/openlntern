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

	lines := make([]string, 0, len(matches)*3+2)
	lines = append(lines, contextPrefix)

	count := 0
	for _, item := range matches {
		uri := strings.TrimSpace(item.URI)
		if uri == "" {
			continue
		}

		// L1: .overview.md - 直接过滤
		if item.Level == 1 || strings.HasSuffix(uri, "/.overview.md") {
			continue
		}

		kbName := strings.TrimSpace(item.KnowledgeBaseName)
		if kbName == "" {
			kbName = "未命名知识库"
		}

		count++
		if item.Level == 0 || strings.HasSuffix(uri, "/abstract.md") || strings.HasSuffix(uri, "/.abstract.md") {
			// L0: abstract.md or .abstract.md
			// 只有 L0 的 abstract 可以被拼接到提示词中，L0 的 URL 不放入。
			// 提供同级的 L1 路径供 Agent 按需读取。
			abstract := strings.TrimSpace(item.Abstract)
			if abstract == "" {
				abstract = "(无摘要)"
			}

			var l1URI string
			if strings.HasSuffix(uri, "/abstract.md") {
				l1URI = strings.Replace(uri, "/abstract.md", "/.overview.md", 1)
			} else if strings.HasSuffix(uri, "/.abstract.md") {
				l1URI = strings.Replace(uri, "/.abstract.md", "/.overview.md", 1)
			} else {
				// 如果是通过 Level=0 命中但没有标准后缀，尝试在同级寻找 .overview.md
				l1URI = uri[:strings.LastIndex(uri, "/")+1] + ".overview.md"
			}

			lines = append(lines, fmt.Sprintf("%d. [知识库:%s][概览摘要]", count, kbName))
			lines = append(lines, "摘要内容: "+abstract)
			// L0 的路径一律不加进去，而是把 L0 同级下的 L1 路径加进去，作为他的进一步大纲内容
			lines = append(lines, "进一步概览/大纲 URI (L1): "+l1URI)
		} else {
			// L2: 具体内容文件
			// 对于 L2 级内容，只放 URL，Agent 决定是否具体访问。
			lines = append(lines, fmt.Sprintf("%d. [知识库:%s][具体资源]", count, kbName))
			lines = append(lines, "uri: "+uri)
		}
	}

	if count == 0 {
		return ""
	}

	// 告知 Agent 如何读取更多信息
	lines = append(lines, "\n提示：如果 [概览摘要] 不足以回答问题，你可以根据提供的 L1 URI 读取概览信息，或根据 [具体资源] 的 URI 读取详细内容。请使用 read_kb_retry 工具。")

	return strings.Join(lines, "\n")
}
