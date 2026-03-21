package toolsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"openIntern/internal/util"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const visibleDynamicToolNamesRunLocalKey = "openintern_visible_dynamic_tool_names"

// VisibilityMiddleware 负责将动态工具注入执行层，并在模型侧按需显隐。
type VisibilityMiddleware struct {
	adk.BaseChatModelAgentMiddleware

	dynamicTools             []einoTool.BaseTool
	dynamicToolNames         map[string]struct{}
	initialVisibleToolNames  []string
	allowToolSearchSelection bool
}

// modelWrapper 在每次模型调用时过滤动态工具的可见性。
type modelWrapper struct {
	allTools                []*schema.ToolInfo
	dynamicToolNames        map[string]struct{}
	initialVisibleToolNames []string
	model                   model.BaseChatModel
}

// NewVisibilityMiddleware 创建按工具名控制可见性的运行时 handler。
func NewVisibilityMiddleware(ctx context.Context, dynamicTools []einoTool.BaseTool, initialVisibleToolNames []string, allowToolSearchSelection bool) (adk.ChatModelAgentMiddleware, error) {
	dynamicToolNames, err := collectToolNames(ctx, dynamicTools)
	if err != nil {
		return nil, err
	}
	return &VisibilityMiddleware{
		dynamicTools:             append([]einoTool.BaseTool{}, dynamicTools...),
		dynamicToolNames:         sliceToSet(dynamicToolNames),
		initialVisibleToolNames:  util.NormalizeUniqueStringList(initialVisibleToolNames),
		allowToolSearchSelection: allowToolSearchSelection,
	}, nil
}

func (m *VisibilityMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	if runCtx == nil || len(m.dynamicTools) == 0 {
		return ctx, runCtx, nil
	}

	next := *runCtx
	next.Tools = append(append([]einoTool.BaseTool{}, runCtx.Tools...), m.dynamicTools...)
	return ctx, &next, nil
}

func (m *VisibilityMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	_ = mc

	visibleToolNames := append([]string{}, m.initialVisibleToolNames...)
	if state != nil && m.allowToolSearchSelection {
		visibleToolNames = append(visibleToolNames, extractSelectedDynamicToolNames(state.Messages)...)
	}
	visibleToolNames = util.NormalizeUniqueStringList(visibleToolNames)
	if err := adk.SetRunLocalValue(ctx, visibleDynamicToolNamesRunLocalKey, visibleToolNames); err != nil {
		return ctx, nil, err
	}
	return ctx, state, nil
}

func (m *VisibilityMiddleware) WrapModel(_ context.Context, cm model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	return &modelWrapper{
		allTools:                mc.Tools,
		dynamicToolNames:        m.dynamicToolNames,
		initialVisibleToolNames: m.initialVisibleToolNames,
		model:                   cm,
	}, nil
}

func (w *modelWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	visibleTools, err := w.resolveVisibleTools(ctx)
	if err != nil {
		return nil, err
	}
	return w.model.Generate(ctx, input, append(opts, model.WithTools(visibleTools))...)
}

func (w *modelWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	visibleTools, err := w.resolveVisibleTools(ctx)
	if err != nil {
		return nil, err
	}
	return w.model.Stream(ctx, input, append(opts, model.WithTools(visibleTools))...)
}

func (w *modelWrapper) resolveVisibleTools(ctx context.Context) ([]*schema.ToolInfo, error) {
	visibleDynamicToolNames, err := getVisibleDynamicToolNames(ctx, w.initialVisibleToolNames)
	if err != nil {
		return nil, err
	}

	filtered := make([]*schema.ToolInfo, 0, len(w.allTools))
	for _, info := range w.allTools {
		if info == nil {
			continue
		}
		name := strings.TrimSpace(info.Name)
		if name == "" {
			continue
		}
		if _, isDynamic := w.dynamicToolNames[name]; isDynamic {
			if _, isVisible := visibleDynamicToolNames[name]; !isVisible {
				continue
			}
		}
		filtered = append(filtered, info)
	}
	return filtered, nil
}

func getVisibleDynamicToolNames(ctx context.Context, fallback []string) (map[string]struct{}, error) {
	value, found, err := adk.GetRunLocalValue(ctx, visibleDynamicToolNamesRunLocalKey)
	if err != nil {
		return nil, err
	}
	if !found {
		return sliceToSet(fallback), nil
	}

	switch typed := value.(type) {
	case []string:
		return sliceToSet(typed), nil
	case []any:
		names := make([]string, 0, len(typed))
		for _, item := range typed {
			name := strings.TrimSpace(fmt.Sprintf("%v", item))
			if name != "" {
				names = append(names, name)
			}
		}
		return sliceToSet(util.NormalizeUniqueStringList(names)), nil
	default:
		return nil, fmt.Errorf("unexpected visible dynamic tool names type: %T", value)
	}
}

func extractSelectedDynamicToolNames(messages []*schema.Message) []string {
	selectedToolNames := make([]string, 0)
	for _, message := range messages {
		if message == nil || message.Role != schema.Tool || strings.TrimSpace(message.ToolName) != ToolName {
			continue
		}
		if strings.TrimSpace(message.Content) == "" {
			continue
		}

		var result Result
		if err := json.Unmarshal([]byte(message.Content), &result); err != nil {
			log.Printf("RunAgent ignore malformed tool_search result err=%v", err)
			continue
		}
		selectedToolNames = append(selectedToolNames, result.SelectedToolNames...)
	}
	return util.NormalizeUniqueStringList(selectedToolNames)
}

func collectToolNames(ctx context.Context, tools []einoTool.BaseTool) ([]string, error) {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, err
		}
		if info == nil {
			return nil, fmt.Errorf("tool info is required")
		}
		name := strings.TrimSpace(info.Name)
		if name == "" {
			return nil, fmt.Errorf("tool name is required")
		}
		names = append(names, name)
	}
	return util.NormalizeUniqueStringList(names), nil
}

func sliceToSet(values []string) map[string]struct{} {
	normalized := util.NormalizeUniqueStringList(values)
	result := make(map[string]struct{}, len(normalized))
	for _, value := range normalized {
		result[value] = struct{}{}
	}
	return result
}
