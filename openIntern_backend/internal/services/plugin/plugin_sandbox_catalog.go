package plugin

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"openIntern/internal/config"
	"openIntern/internal/models"
	sandboxsvc "openIntern/internal/services/sandbox"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	builtinSandboxPluginName        = "AIO Sandbox"
	builtinSandboxPluginDescription = "AIO sandbox 工具集合。工具元数据在启动时从 probe 实例抓取，真正执行时按当前用户懒创建并连接其隔离 sandbox。"
	builtinSandboxProbeUserID       = "openintern-builtin-sandbox-catalog"
)

var builtinSandboxToolNames = map[string]struct{}{}

// InitBuiltinSandboxCatalog discovers the current AIO sandbox tool catalog once at startup
// and exposes the result as a single builtin plugin definition shared by all users.
func InitBuiltinSandboxCatalog(cfg config.SandboxConfig) error {
	if cfg.Enabled == nil || !*cfg.Enabled {
		builtinSandboxToolNames = map[string]struct{}{}
		return nil
	}

	definition, err := loadBuiltinSandboxPluginDefinition(cfg)
	if err != nil {
		return err
	}

	builtinPluginDefinitions = mergeBuiltinPluginDefinitions(builtinPluginDefinitions, definition)
	toolNames := make(map[string]struct{}, len(definition.tools))
	for _, tool := range definition.tools {
		toolName := strings.TrimSpace(tool.ToolName)
		if toolName == "" {
			continue
		}
		toolNames[toolName] = struct{}{}
	}
	builtinSandboxToolNames = toolNames
	log.Printf("builtin sandbox catalog loaded tools=%d", len(definition.tools))
	return nil
}

func loadBuiltinSandboxPluginDefinition(cfg config.SandboxConfig) (builtinPluginDefinition, error) {
	provider, err := sandboxsvc.BuildProvider(cfg)
	if err != nil {
		return builtinPluginDefinition{}, err
	}

	createTimeout := time.Duration(cfg.CreateTimeoutSeconds) * time.Second
	if createTimeout <= 0 {
		createTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), createTimeout+20*time.Second)
	defer cancel()

	result, err := provider.Create(ctx, sandboxsvc.CreateRequest{UserID: builtinSandboxProbeUserID})
	if err != nil {
		return builtinPluginDefinition{}, fmt.Errorf("create sandbox probe failed: %w", err)
	}
	instance := sandboxsvc.Instance{
		UserID:     builtinSandboxProbeUserID,
		Provider:   provider.Name(),
		InstanceID: result.InstanceID,
		Endpoint:   result.Endpoint,
	}
	defer func() {
		if destroyErr := provider.Destroy(context.Background(), instance); destroyErr != nil {
			log.Printf("destroy sandbox probe failed instance_id=%s err=%v", instance.InstanceID, destroyErr)
		}
	}()

	remoteTools, err := fetchMCPToolDefinitions(ctx, strings.TrimRight(result.Endpoint, "/")+"/mcp", mcpProtocolStreamableHTTP)
	if err != nil {
		remoteTools, err = fetchSandboxToolDefinitionsWithRetry(ctx, strings.TrimRight(result.Endpoint, "/")+"/mcp")
	}
	if err != nil {
		return builtinPluginDefinition{}, fmt.Errorf("fetch sandbox tool definitions failed: %w", err)
	}
	if len(remoteTools) == 0 {
		return builtinPluginDefinition{}, fmt.Errorf("sandbox probe returned no tools")
	}

	return buildBuiltinSandboxPluginDefinition(remoteTools)
}

func buildBuiltinSandboxPluginDefinition(remoteTools []mcp.Tool) (builtinPluginDefinition, error) {
	pluginID := stableBuiltinSandboxUUID("plugin")
	defaultOutputSchema, err := normalizeOutputSchema("")
	if err != nil {
		return builtinPluginDefinition{}, err
	}

	tools := make([]models.Tool, 0, len(remoteTools))
	seenToolNames := make(map[string]struct{}, len(remoteTools))
	for _, remoteTool := range remoteTools {
		toolName := strings.TrimSpace(remoteTool.Name)
		if toolName == "" {
			continue
		}
		if _, exists := seenToolNames[toolName]; exists {
			return builtinPluginDefinition{}, fmt.Errorf("duplicate sandbox tool_name: %s", toolName)
		}
		seenToolNames[toolName] = struct{}{}

		inputSchemaJSON, err := marshalMCPToolSchema(remoteTool.InputSchema)
		if err != nil {
			return builtinPluginDefinition{}, fmt.Errorf("marshal sandbox input schema for %s failed: %w", toolName, err)
		}

		outputSchemaJSON := defaultOutputSchema
		if hasMCPOutputSchema(remoteTool) {
			outputSchemaJSON, err = marshalMCPToolSchema(remoteTool.OutputSchema)
			if err != nil {
				return builtinPluginDefinition{}, fmt.Errorf("marshal sandbox output schema for %s failed: %w", toolName, err)
			}
		}

		tools = append(tools, models.Tool{
			ToolID:      stableBuiltinSandboxUUID("tool:" + toolName),
			PluginID:    pluginID,
			ToolName:    toolName,
			Description: strings.TrimSpace(remoteTool.Description),
			// sandbox builtin 工具默认直接可见；后续若要走 tool_search，可在这里注入 lazy/search_hint。
			LazyLoad:         false,
			SearchHint:       "",
			InputSchemaJSON:  inputSchemaJSON,
			OutputSchemaJSON: outputSchemaJSON,
			ToolResponseMode: toolResponseNonStreaming,
			Enabled:          true,
			TimeoutMS:        defaultPluginTimeoutMS,
		})
	}

	if len(tools) == 0 {
		return builtinPluginDefinition{}, fmt.Errorf("sandbox probe returned no valid tools")
	}

	return builtinPluginDefinition{
		plugin: models.Plugin{
			PluginID:    pluginID,
			Name:        builtinSandboxPluginName,
			Description: builtinSandboxPluginDescription,
			Source:      pluginSourceBuiltin,
			RuntimeType: pluginRuntimeBuiltin,
			Status:      pluginStatusEnabled,
			LazyLoad:    false,
		},
		tools: tools,
	}, nil
}

func mergeBuiltinPluginDefinitions(existing []builtinPluginDefinition, incoming builtinPluginDefinition) []builtinPluginDefinition {
	merged := make([]builtinPluginDefinition, 0, len(existing)+1)
	replaced := false
	for _, item := range existing {
		if item.plugin.PluginID == incoming.plugin.PluginID || strings.EqualFold(strings.TrimSpace(item.plugin.Name), strings.TrimSpace(incoming.plugin.Name)) {
			merged = append(merged, incoming)
			replaced = true
			continue
		}
		merged = append(merged, item)
	}
	if !replaced {
		merged = append(merged, incoming)
	}
	return merged
}

func isBuiltinSandboxToolName(toolName string) bool {
	_, ok := builtinSandboxToolNames[strings.TrimSpace(toolName)]
	return ok
}

func stableBuiltinSandboxUUID(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("openintern/builtin/sandbox/"+strings.TrimSpace(name))).String()
}

func fetchSandboxToolDefinitionsWithRetry(ctx context.Context, mcpURL string) ([]mcp.Tool, error) {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				if lastErr != nil {
					return nil, lastErr
				}
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}

		tools, err := fetchMCPToolDefinitions(ctx, mcpURL, mcpProtocolStreamableHTTP)
		if err == nil {
			return tools, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("sandbox mcp tools are not ready")
}
