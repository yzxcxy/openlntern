package builtin_tool

import (
	"context"
	"errors"
	"strings"

	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	sandboxBuiltinToolName       = "sandbox_execute_bash"
	sandboxMCPProtocolSSE        = "sse"
	sandboxMCPProtocolStreamHTTP = "streamableHttp"
)

// GetSandboxMCPTools 从 sandbox MCP 中拉取固定内建工具。
func GetSandboxMCPTools(ctx context.Context, baseURL string) ([]einoTool.BaseTool, func(), error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, nil, errors.New("sandbox base url not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	mcpURL := strings.TrimRight(baseURL, "/") + "/mcp"
	cli, err := client.NewStreamableHttpClient(mcpURL)
	if err != nil {
		return nil, nil, err
	}
	if err := cli.Start(ctx); err != nil {
		_ = cli.Close()
		return nil, nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "openintern",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		_ = cli.Close()
		return nil, nil, err
	}

	tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{
		Cli:          cli,
		ToolNameList: []string{sandboxBuiltinToolName},
	})
	if err != nil {
		_ = cli.Close()
		return nil, nil, err
	}
	if len(tools) == 0 {
		_ = cli.Close()
		return nil, nil, errors.New("sandbox_execute_bash tool not found")
	}

	cleanup := func() {
		_ = cli.Close()
	}
	return tools, cleanup, nil
}
