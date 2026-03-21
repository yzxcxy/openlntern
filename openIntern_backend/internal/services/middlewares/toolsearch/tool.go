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
	ToolName             = "tool_search"
	defaultRequestedTopK = 4
	maxRequestedTopK     = 8
	intentParamDesc      = "描述你当前需要的工具能力，例如：查询襄阳实时天气、读取网页内容、搜索 Go 1.26 发布说明。不要直接复制整段用户原话。"
	keywordsParamDesc    = "补充关键实体词列表，例如城市名、产品名、版本号。只有在 intent 里没有明确覆盖时再传。"
	topKParamDesc        = "期望加载的工具数量，建议保持较小值，范围 1-8，默认 4。"
)

// Input 定义给模型暴露的最小检索参数集合。
type Input struct {
	Intent   string   `json:"intent" jsonschema_description:"描述当前需要的工具能力。"`
	Keywords []string `json:"keywords,omitempty" jsonschema_description:"补充实体词或关键词。"`
	TopK     int      `json:"top_k,omitempty" jsonschema_description:"限制希望返回的工具数量，范围 1-8。"`
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
		"搜索插件工具库，并在后续推理中加载最相关的工具。",
		"当你当前看不到合适工具，但明确需要外部能力时使用。",
		"intent: " + intentParamDesc,
		"keywords: " + keywordsParamDesc,
		"top_k: " + topKParamDesc,
	}, " ")
}

func runToolSearch(ctx context.Context, input Input) (string, error) {
	query, err := buildQuery(input)
	if err != nil {
		return "", err
	}

	matches, err := plugin.Plugin.SearchRuntimeTools(ctx, query, plugin.ToolSearchOptions{
		TopK: normalizeRequestedTopK(input.TopK),
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

func buildQuery(input Input) (string, error) {
	intent := strings.TrimSpace(input.Intent)
	if intent == "" {
		return "", fmt.Errorf("intent is required")
	}

	queryParts := []string{intent}
	keywords := util.NormalizeUniqueStringList(input.Keywords)
	if len(keywords) > 0 {
		queryParts = append(queryParts, strings.Join(keywords, " "))
	}
	return strings.Join(queryParts, "\n"), nil
}

func normalizeRequestedTopK(value int) int {
	if value <= 0 {
		return defaultRequestedTopK
	}
	if value > maxRequestedTopK {
		return maxRequestedTopK
	}
	return value
}
