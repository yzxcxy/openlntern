package builtin_tool

import (
	"context"
	"errors"
	"fmt"
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
	cli, err := openSandboxMCPClient(ctx, baseURL)
	if err != nil {
		return nil, nil, err
	}

	tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{
		Cli:          cli,
		ToolNameList: []string{sandboxBuiltinToolName},
	})
	if err != nil {
		_ = cli.Close()
		return nil, nil, fmt.Errorf("load sandbox mcp tools failed: %w", err)
	}
	if len(tools) == 0 {
		_ = cli.Close()
		return nil, nil, errors.New("builtin sandbox tool sandbox_execute_bash not found")
	}

	cleanup := func() {
		if err := cli.Close(); err != nil {
			_ = err
		}
	}
	return tools, cleanup, nil
}

// openSandboxMCPClient 允许使用 tools.sandbox.url 自动探测 sandbox MCP 协议。
func openSandboxMCPClient(ctx context.Context, baseURL string) (*client.Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.New("sandbox base url not configured")
	}

	var errs []error
	for _, protocol := range []string{sandboxMCPProtocolStreamHTTP, sandboxMCPProtocolSSE} {
		cli, err := openSandboxMCPClientWithProtocol(ctx, baseURL, protocol)
		if err == nil {
			return cli, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", protocol, err))
	}

	return nil, fmt.Errorf("connect sandbox mcp server failed: %w", errors.Join(errs...))
}

func openSandboxMCPClientWithProtocol(ctx context.Context, baseURL string, protocol string) (*client.Client, error) {
	var (
		cli *client.Client
		err error
	)

	switch protocol {
	case sandboxMCPProtocolSSE:
		cli, err = client.NewSSEMCPClient(baseURL)
	case sandboxMCPProtocolStreamHTTP:
		cli, err = client.NewStreamableHttpClient(baseURL)
	default:
		return nil, fmt.Errorf("unsupported sandbox mcp protocol: %s", protocol)
	}
	if err != nil {
		return nil, err
	}

	if err := cli.Start(ctx); err != nil {
		_ = cli.Close()
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "openintern-builtin",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		_ = cli.Close()
		return nil, err
	}

	return cli, nil
}
