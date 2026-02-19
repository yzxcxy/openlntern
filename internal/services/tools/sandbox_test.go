package tools

import (
	"context"
	"testing"
	"time"
)

func TestGetSandboxTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tools, err := GetSandboxTools(ctx, "http://localhost:8081/mcp")

	if err != nil {
		t.Logf("get sandbox tools failed: %v", err)	
	}

	// 打印所有工具的信息
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		if err != nil {
			t.Logf("tool info failed: %v", err)
			continue
		}
		t.Logf("tool: %s, desc: %s", info.Name, info.Desc)
	}
}
