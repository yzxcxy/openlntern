package builtin_tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"openIntern/internal/dao"
	kbsvc "openIntern/internal/services/kb"
	"openIntern/internal/util"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

var (
	errKBEntryURIRequired  = errors.New("uri is required")
	errKBEntryOutOfScope   = errors.New("uri does not belong to the current knowledge bases")
	errKnowledgeBaseAbsent = errors.New("knowledge base is not available in this context")
)

// ReadKnowledgeBaseEntryInput 表示读取知识库正文的工具入参。
type ReadKnowledgeBaseEntryInput struct {
	URI string `json:"uri" jsonschema_description:"知识库条目的完整 URI，通常来自当前回合知识库召回结果中的 uri 字段"`
}

// GetKnowledgeBaseTools 返回受当前知识库绑定范围约束的运行时工具。
func GetKnowledgeBaseTools(ctx context.Context, knowledgeBaseNames []string) ([]einoTool.BaseTool, error) {
	_ = ctx
	allowedPrefixes := scopedKnowledgeBasePrefixes(knowledgeBaseNames)
	if len(allowedPrefixes) == 0 {
		return nil, nil
	}
	// 工具闭包持有当前会话允许访问的知识库前缀，避免变成全局任意资源读取入口。
	readTool, err := utils.InferTool[ReadKnowledgeBaseEntryInput, string](
		"read_kb_retry",
		"根据知识库召回结果中的 uri 读取该条目的完整内容。仅允许读取当前会话已绑定知识库下的条目。",
		func(ctx context.Context, input ReadKnowledgeBaseEntryInput) (string, error) {
			return readKnowledgeBaseEntryImpl(ctx, input, allowedPrefixes)
		},
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{readTool}, nil
}

func readKnowledgeBaseEntryImpl(ctx context.Context, input ReadKnowledgeBaseEntryInput, allowedPrefixes []string) (string, error) {
	uri := strings.TrimSpace(input.URI)
	if uri == "" {
		return "", errKBEntryURIRequired
	}
	if len(allowedPrefixes) == 0 {
		return "", errKnowledgeBaseAbsent
	}
	if !uriAllowedInKnowledgeBases(uri, allowedPrefixes) {
		return "", errKBEntryOutOfScope
	}
	content, err := kbsvc.KnowledgeBase.ReadContent(ctx, uri)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(map[string]string{
		"uri":     uri,
		"content": content,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func scopedKnowledgeBasePrefixes(knowledgeBaseNames []string) []string {
	names := util.NormalizeUniqueStringList(knowledgeBaseNames)
	prefixes := make([]string, 0, len(names))
	for _, name := range names {
		cleanName, err := dao.KnowledgeBase.CleanName(name)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, dao.KnowledgeBase.URI(cleanName))
	}
	return util.NormalizeUniqueStringList(prefixes)
}

func uriAllowedInKnowledgeBases(uri string, allowedPrefixes []string) bool {
	trimmedURI := strings.TrimSpace(uri)
	if trimmedURI == "" {
		return false
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(trimmedURI, prefix) {
			return true
		}
	}
	return false
}
