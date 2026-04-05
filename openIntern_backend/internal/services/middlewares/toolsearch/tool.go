package toolsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"openIntern/internal/services/plugin"
	"openIntern/internal/util"
	"strings"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	ToolName                  = "tool_search"
	defaultRequestedMaxResult = 5
	maxRequestedMaxResult     = 8
	queryParamDesc            = "用于查找工具的查询字符串。可使用 select:tool_name 直接选择工具，或输入关键词进行搜索。"
	maxResultsParamDesc       = "期望返回的工具数量，范围 1-8，默认 5。"
)

// Input 定义给模型暴露的 Claude Code 风格检索参数集合。
type Input struct {
	Query      string `json:"query" jsonschema_description:"Query to find deferred tools. Use select:tool_name for direct selection, or keywords to search."`
	MaxResults int    `json:"max_results,omitempty" jsonschema_description:"Maximum number of results to return, between 1 and 8."`
}

// Result 仅保留后续可见性控制所需的最小工具名列表。
type Result struct {
	SelectedToolNames []string `json:"selected_tool_names"`
}

// NewTool 创建供模型调用的语义工具检索元工具。
func NewTool(_ context.Context) (einoTool.BaseTool, error) {
	return utils.InferTool[Input, string](
		ToolName,
		buildToolDescription(),
		func(ctx context.Context, input Input) (string, error) {
			return runToolSearch(ctx, input)
		},
	)
}

func buildToolDescription() string {
	return strings.Join([]string{
		"获取当前延迟加载工具中最相关的工具定义，供后续调用。",
		"当你需要的工具当前不可见时使用。",
		`支持三种查询形式：select:tool_a,tool_b；普通关键词检索；以及 +required optional 形式的必选词搜索。`,
		"query: " + queryParamDesc,
		"max_results: " + maxResultsParamDesc,
	}, " ")
}

func runToolSearch(ctx context.Context, input Input) (string, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	matches, err := plugin.Plugin.SearchRuntimeTools(ctx, query, plugin.ToolSearchOptions{
		TopK: normalizeRequestedMaxResults(input.MaxResults),
	})
	if err != nil {
		return "", err
	}

	names := make([]string, 0, len(matches))
	for _, match := range matches {
		name := strings.TrimSpace(match.ToolName)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	result := Result{
		SelectedToolNames: util.NormalizeUniqueStringList(names),
	}
	output, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func normalizeRequestedMaxResults(value int) int {
	if value <= 0 {
		return defaultRequestedMaxResult
	}
	if value > maxRequestedMaxResult {
		return maxRequestedMaxResult
	}
	return value
}
