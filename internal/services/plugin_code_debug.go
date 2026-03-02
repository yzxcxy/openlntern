package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	toolsvc "openIntern/internal/services/tools"

	einoTool "github.com/cloudwego/eino/components/tool"
)

type CodeDebugInput struct {
	Code             string             `json:"code"`
	CodeLanguage     string             `json:"code_language"`
	BodyFields       []PluginFieldInput `json:"body_fields"`
	OutputSchemaJSON string             `json:"output_schema_json"`
	Input            map[string]any     `json:"input"`
	TimeoutMS        int                `json:"timeout_ms"`
}

type CodeDebugResult struct {
	Result    any    `json:"result"`
	RawOutput string `json:"raw_output"`
}

func (s *PluginService) DebugCodeTool(ctx context.Context, input CodeDebugInput) (CodeDebugResult, error) {
	if strings.TrimSpace(sandboxBaseURL) == "" {
		return CodeDebugResult{}, errors.New("sandbox base url not configured")
	}

	tool, err := s.buildTool("", pluginRuntimeCode, PluginToolInput{
		ToolName:         "code_debug",
		Description:      "debug code tool",
		ToolResponseMode: toolResponseNonStreaming,
		OutputSchemaJSON: input.OutputSchemaJSON,
		BodyFields:       input.BodyFields,
		TimeoutMS:        input.TimeoutMS,
		Code:             input.Code,
		CodeLanguage:     input.CodeLanguage,
	})
	if err != nil {
		return CodeDebugResult{}, err
	}

	runtimeTool, err := toolsvc.NewCodePluginTool(tool)
	if err != nil {
		return CodeDebugResult{}, err
	}

	invokable, ok := runtimeTool.(einoTool.InvokableTool)
	if !ok {
		return CodeDebugResult{}, errors.New("code debug tool is not invokable")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, toolsvc.ContextKeySandboxBaseURL, sandboxBaseURL)

	arguments, err := json.Marshal(input.Input)
	if err != nil {
		return CodeDebugResult{}, err
	}

	output, err := invokable.InvokableRun(ctx, string(arguments))
	if err != nil {
		return CodeDebugResult{}, err
	}

	result := any(strings.TrimSpace(output))
	if parsed, parseErr := parseCodeDebugOutput(output); parseErr == nil {
		result = parsed
	}

	return CodeDebugResult{
		Result:    result,
		RawOutput: output,
	}, nil
}

func parseCodeDebugOutput(raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty output")
	}

	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
