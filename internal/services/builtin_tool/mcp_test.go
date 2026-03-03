package builtin_tool

import (
	"context"
	"testing"

	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCP(t *testing.T) {
	ctx := context.Background()
	baseURL := "http://localhost:8081/mcp"
	cli, err := client.NewStreamableHttpClient(baseURL)
	if err := cli.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}
	tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{Cli: cli})
	if err != nil {
		t.Fatalf("Failed to get tools: %v", err)
	}
	if len(tools) > 0 {
		info, err := tools[0].Info(ctx)
		if err != nil {
			t.Fatalf("Failed to get tool info: %v", err)
		}
		t.Logf("tools: %v", info)
	}
}
