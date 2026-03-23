package agent

import (
	"context"
	"fmt"
	"strings"

	"openIntern/internal/util"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// renamedInvokableTool 仅重写暴露给模型的 tool name，执行逻辑保持委托给底层 tool。
type renamedInvokableTool struct {
	base tool.InvokableTool
	info *schema.ToolInfo
}

func newRenamedInvokableTool(ctx context.Context, base tool.InvokableTool, safeName string) (tool.BaseTool, error) {
	safeName = normalizeSubAgentToolName(safeName)
	if !util.IsModelSafeToolName(safeName) {
		return nil, fmt.Errorf("tool name must match ^[a-zA-Z0-9_-]+$: %s", safeName)
	}

	info, err := base.Info(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("tool info is required")
	}

	return &renamedInvokableTool{
		base: base,
		info: cloneToolInfoWithName(info, safeName),
	}, nil
}

func (t *renamedInvokableTool) Info(context.Context) (*schema.ToolInfo, error) {
	return cloneToolInfoWithName(t.info, t.info.Name), nil
}

func (t *renamedInvokableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.base.InvokableRun(ctx, argumentsInJSON, opts...)
}

func cloneToolInfoWithName(info *schema.ToolInfo, name string) *schema.ToolInfo {
	if info == nil {
		return nil
	}
	return &schema.ToolInfo{
		Name:        name,
		Desc:        info.Desc,
		ParamsOneOf: info.ParamsOneOf,
	}
}

func buildSubAgentToolName(agentID string) string {
	return normalizeSubAgentToolName("sub_agent_" + strings.TrimSpace(agentID))
}

func normalizeSubAgentToolName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "sub_agent"
	}
	if util.IsModelSafeToolName(raw) {
		return raw
	}

	var builder strings.Builder
	builder.Grow(len(raw))
	lastWasSeparator := false
	for _, r := range raw {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastWasSeparator = false
		case r == '_' || r == '-':
			builder.WriteRune(r)
			lastWasSeparator = false
		default:
			if builder.Len() > 0 && !lastWasSeparator {
				builder.WriteByte('_')
				lastWasSeparator = true
			}
		}
	}

	name := strings.Trim(builder.String(), "_-")
	if name == "" {
		return "sub_agent"
	}
	return name
}
