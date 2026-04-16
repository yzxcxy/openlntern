package builtin_tool

import (
	"context"
	"encoding/json"
	"errors"
	"openIntern/internal/services/sandbox"
	"strings"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SandboxExecuteBashInput struct {
	Cmd string `json:"cmd" jsonschema_description:"要在当前用户 sandbox 中执行的 bash 命令"`
}

func (i *SandboxExecuteBashInput) UnmarshalJSON(data []byte) error {
	type alias SandboxExecuteBashInput
	var payload struct {
		alias
		Command string `json:"command"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	i.Cmd = strings.TrimSpace(payload.alias.Cmd)
	if i.Cmd == "" {
		// 历史线程里可能仍保留旧字段名 command，这里只做一次入参兼容。
		i.Cmd = strings.TrimSpace(payload.Command)
	}
	return nil
}

func sandboxExecuteBashImpl(ctx context.Context, input SandboxExecuteBashInput) (string, error) {
	command := strings.TrimSpace(input.Cmd)
	if command == "" {
		return "", errors.New("cmd is required")
	}
	userID, _ := ctx.Value(ContextKeyUserID).(string)
	instance, err := sandbox.Lifecycle.GetOrCreate(ctx, strings.TrimSpace(userID))
	if err != nil {
		return "", err
	}
	output, err := sandbox.Lifecycle.Client().ExecuteBash(ctx, instance.Endpoint, command)
	if err != nil {
		return "", err
	}
	_ = sandbox.Lifecycle.Touch(ctx, strings.TrimSpace(userID))
	return output, nil
}

// GetSandboxExecuteBashTool 返回本地定义的 sandbox bash 代理工具。
func GetSandboxExecuteBashTool(ctx context.Context) (einoTool.BaseTool, error) {
	return utils.InferTool[SandboxExecuteBashInput, string](
		"sandbox_execute_bash",
		"在当前用户的 AIO sandbox 中执行 bash 命令。",
		sandboxExecuteBashImpl,
	)
}
