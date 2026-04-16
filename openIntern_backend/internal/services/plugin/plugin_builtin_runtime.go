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
	return s.buildRuntimeBuiltinTools(ctx, toolIDs)
}

// BuildAllRuntimeBuiltinTools 构建全部启用态 builtin 插件工具，供动态工具池预装使用。
func (s *PluginService) BuildAllRuntimeBuiltinTools(ctx context.Context) ([]einoTool.BaseTool, error) {
	return s.buildRuntimeBuiltinTools(ctx, nil)
}

func (s *PluginService) buildRuntimeBuiltinTools(ctx context.Context, toolIDs []string) ([]einoTool.BaseTool, error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, nil
	}
	toolRows, err := dao.Plugin.ListRuntimeTools(userID, pluginRuntimeBuiltin, pluginStatusEnabled, toolIDs)
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
			if isBuiltinSandboxToolName(row.ToolName) {
				var err error
				runtimeTool, err = newSandboxBuiltinProxyTool(row)
				if err != nil {
					return nil, fmt.Errorf("build sandbox builtin tool %s failed: %w", row.ToolID, err)
				}
			} else {
				return nil, fmt.Errorf("builtin tool implementation not found: %s", row.ToolName)
			}
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
	objectStorageTools, err := builtinTool.GetObjectStorageTools(ctx)
	if err != nil {
		return nil, err
	}

	catalog := make(map[string]einoTool.BaseTool, len(a2uiTools)+len(objectStorageTools))
	for _, tool := range append(a2uiTools, objectStorageTools...) {
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
