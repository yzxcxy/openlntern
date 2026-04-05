package plugin

import (
	"context"
	"errors"
	"net/http"
	sandboxsvc "openIntern/internal/services/sandbox"
	"strings"
	"time"
)

type CodeDebugInput struct {
	Code         string         `json:"code"`
	CodeLanguage string         `json:"code_language"`
	Input        map[string]any `json:"input"`
	TimeoutMS    int            `json:"timeout_ms"`
}

func (s *PluginService) DebugCodeTool(ctx context.Context, input CodeDebugInput) (any, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return nil, errors.New("code is required")
	}
	codeLanguage := normalizeCodeLanguage(input.CodeLanguage)
	if codeLanguage == "" {
		return nil, errors.New("code_language must be python or javascript")
	}
	timeoutMS := input.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30000
	}
	wrappedCode, err := WrapCodeForMainExecution(codeLanguage, code, input.Input)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	userID := userIDFromContext(ctx)
	instance, err := sandboxsvc.Lifecycle.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	output, err := RunCodeInSandbox(ctx, &http.Client{
		Timeout: time.Duration(timeoutMS) * time.Millisecond,
	}, instance.Endpoint, SandboxCodeRunInput{
		CodeLanguage: codeLanguage,
		Code:         wrappedCode,
		Input:        input.Input,
		TimeoutMS:    timeoutMS,
	})
	if err != nil {
		return nil, err
	}
	_ = sandboxsvc.Lifecycle.Touch(ctx, userID)

	result, err := ParseSandboxCodeExecutionOutput(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}
