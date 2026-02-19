package tools

import (
	"context"
	"strings"

	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func GetSandboxTools(ctx context.Context, baseURL string) ([]einoTool.BaseTool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if baseURL == "" {
		baseURL = "http://localhost:8081/mcp"
	}
	cli, err := client.NewStreamableHttpClient(baseURL)
	if err != nil {
		return nil, err
	}
	if err := cli.Start(ctx); err != nil {
		return nil, err
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "openintern",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		return nil, err
	}
	tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{Cli: cli})
	if err != nil {
		return nil, err
	}
	filtered := make([]einoTool.BaseTool, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		if !excludeTools(info) {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

func excludeTools(info *schema.ToolInfo) bool {
	if info == nil {
		return false
	}
	name := strings.ToLower(info.Name)
	excludes := []string{"file","browser","editor"}
	for _, exclude := range excludes {
		if strings.Contains(name, exclude) {
			return true
		}
	}
	return false
}
