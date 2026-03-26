package plugin

import (
	"context"
	"log"
	"strings"
	"time"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// mcpToolWithTimeout wraps an MCP tool with timeout control and fixes empty argument handling.
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

	// MCP tools with empty input schema expect null instead of {}.
	// Python Pydantic validates {} as "unexpected keyword argument" for empty models.
	// This workaround converts empty object to null for all MCP tool calls.
	originalArgs := argumentsInJSON
	if strings.TrimSpace(argumentsInJSON) == "{}" {
		argumentsInJSON = "null"
		log.Printf("mcpToolWithTimeout: converted empty args '{}' to 'null'")
	}
	if originalArgs != argumentsInJSON {
		log.Printf("mcpToolWithTimeout: args changed from '%s' to '%s'", originalArgs, argumentsInJSON)
	}

	return t.tool.InvokableRun(ctx, argumentsInJSON, opts...)
}