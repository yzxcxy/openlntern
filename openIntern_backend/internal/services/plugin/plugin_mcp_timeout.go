package plugin

import (
	"context"
	"time"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// mcpToolWithTimeout wraps an MCP tool with timeout control.
type mcpToolWithTimeout struct {
	tool      einoTool.InvokableTool
	timeoutMS int
}

var _ einoTool.InvokableTool = (*mcpToolWithTimeout)(nil)

func newMCPToolWithTimeout(tool einoTool.BaseTool, timeoutMS int) einoTool.BaseTool {
	if invokable, ok := tool.(einoTool.InvokableTool); ok {
		return &mcpToolWithTimeout{
			tool:      invokable,
			timeoutMS: timeoutMS,
		}
	}
	return tool
}

func (t *mcpToolWithTimeout) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.tool.Info(ctx)
}

func (t *mcpToolWithTimeout) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einoTool.Option) (string, error) {
	timeout := time.Duration(t.timeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return t.tool.InvokableRun(ctx, argumentsInJSON, opts...)
}