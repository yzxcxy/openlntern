package plugin

import (
	"context"
	"fmt"

	"openIntern/internal/dao"
	builtinTool "openIntern/internal/services/builtin_tool"
	"openIntern/internal/util"

	einoTool "github.com/cloudwego/eino/components/tool"
)

// BuildRuntimeBuiltinTools resolves builtin tool records from the plugin tables and maps them to in-process tool implementations.
func (s *PluginService) BuildRuntimeBuiltinTools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, error) {
	toolIDs = util.NormalizeUniqueStringList(toolIDs)
	if len(toolIDs) == 0 {
		return nil, nil
	}

	toolRows, err := dao.Plugin.ListRuntimeTools(pluginRuntimeBuiltin, pluginStatusEnabled, toolIDs)
	if err != nil {
		return nil, err
	}
	if len(toolRows) == 0 {
		return nil, nil
	}

	catalog, err := loadBuiltinRuntimeToolCatalog(ctx)
	if err != nil {
		return nil, err
	}

	runtimeTools := make([]einoTool.BaseTool, 0, len(toolRows))
	for _, row := range toolRows {
		runtimeTool, ok := catalog[row.ToolName]
		if !ok {
			return nil, fmt.Errorf("builtin tool implementation not found: %s", row.ToolName)
		}
		runtimeTools = append(runtimeTools, runtimeTool)
	}
	return runtimeTools, nil
}

func loadBuiltinRuntimeToolCatalog(ctx context.Context) (map[string]einoTool.BaseTool, error) {
	a2uiTools, err := builtinTool.GetA2UITools(ctx)
	if err != nil {
		return nil, err
	}
	cosTools, err := builtinTool.GetCOSTools(ctx)
	if err != nil {
		return nil, err
	}

	catalog := make(map[string]einoTool.BaseTool, len(a2uiTools)+len(cosTools))
	for _, tool := range append(a2uiTools, cosTools...) {
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, err
		}
		if info == nil || info.Name == "" {
			return nil, fmt.Errorf("builtin tool info is required")
		}
		catalog[info.Name] = tool
	}
	return catalog, nil
}
