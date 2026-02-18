package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"openIntern/internal/agui"
	"openIntern/internal/config"

	"github.com/cloudwego/eino/schema"
)

func TestRunEinoConversation(t *testing.T) {
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate test file path")
	}
	backendDir := filepath.Dir(filepath.Dir(filepath.Dir(filePath)))
	cfgPath := filepath.Join(backendDir, "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("config not found: %v", err)
	}
	cfg := config.LoadConfig(cfgPath)
	if cfg.LLM.APIKey == "" || cfg.LLM.Model == "" {
		t.Skip("llm config missing")
	}
	if err := InitEino(cfg.LLM); err != nil {
		t.Fatalf("init eino failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	messages := []*schema.Message{
		schema.SystemMessage("你是一个简洁助手"),
		schema.UserMessage("你好"),
		schema.AssistantMessage("你好！有什么可以帮你？", nil),
		schema.UserMessage("请用一句话解释Go的协程"),
	}
	acc := agui.NewAccumulator("thread-test")
	baseSender := agui.NewSenderWithThreadID(ctx, io.Discard, "thread-test")
	sender := agui.NewAccumulatingSender(baseSender, acc)
	if err := sender.Start(); err != nil {
		t.Fatalf("start sender failed: %v", err)
	}
	err := runEinoStreaming(ctx, sender, messages)
	if finishErr := sender.Finish(); finishErr != nil {
		t.Fatalf("finish sender failed: %v", finishErr)
	}
	if err != nil {
		t.Fatalf("runEinoStreaming failed: %v", err)
	}
	flushed := acc.Flush()
	found := false
	for _, msg := range flushed {
		if msg.Type != "text" {
			continue
		}
		if content, ok := msg.Content.(string); ok && content != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("empty assistant response")
	}

	t.Log(flushed)
}
