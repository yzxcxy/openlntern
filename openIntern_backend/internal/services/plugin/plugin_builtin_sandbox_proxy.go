package plugin

import (
	"context"
	"fmt"
	"strings"

	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"openIntern/internal/models"
	sandboxsvc "openIntern/internal/services/sandbox"
)

type sandboxBuiltinProxyTool struct {
	def  models.Tool
	info *schema.ToolInfo
}

var _ einoTool.InvokableTool = (*sandboxBuiltinProxyTool)(nil)

func newSandboxBuiltinProxyTool(def models.Tool) (einoTool.BaseTool, error) {
	info, err := buildCodePluginToolInfo(def)
	if err != nil {
		return nil, err
	}
	return &sandboxBuiltinProxyTool{
		def:  def,
		info: info,
	}, nil
}

func (t *sandboxBuiltinProxyTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *sandboxBuiltinProxyTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einoTool.Option) (string, error) {
	_ = opts

	userID := userIDFromContext(ctx)
	if userID == "" {
		return "", fmt.Errorf("user_id is required")
	}

	instance, err := sandboxsvc.Lifecycle.GetOrCreate(ctx, userID)
	if err != nil {
		return "", err
	}

	cli, err := openMCPClient(ctx, strings.TrimRight(instance.Endpoint, "/")+"/mcp", mcpProtocolStreamableHTTP)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = cli.Close()
	}()

	tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{
		Cli:          cli,
		ToolNameList: []string{t.def.ToolName},
	})
	if err != nil {
		return "", fmt.Errorf("load sandbox builtin tool %s failed: %w", t.def.ToolName, err)
	}
	if len(tools) == 0 {
		return "", fmt.Errorf("sandbox builtin tool %s not found", t.def.ToolName)
	}

	invokable, ok := newMCPToolWithTimeout(tools[0], t.def.TimeoutMS).(einoTool.InvokableTool)
	if !ok {
		return "", fmt.Errorf("sandbox builtin tool %s is not invokable", t.def.ToolName)
	}
	result, err := invokable.InvokableRun(ctx, argumentsInJSON)
	if err != nil {
		return "", err
	}
	_ = sandboxsvc.Lifecycle.Touch(ctx, userID)
	return result, nil
}
