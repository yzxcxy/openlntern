package plugin

import (
	"context"
	"errors"
	"net/http"
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
	if strings.TrimSpace(sandboxBaseURL) == "" {
		return nil, errors.New("sandbox base url not configured")
	}

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
	output, err := RunCodeInSandbox(ctx, &http.Client{
		Timeout: time.Duration(timeoutMS) * time.Millisecond,
	}, sandboxBaseURL, SandboxCodeRunInput{
		CodeLanguage: codeLanguage,
		Code:         wrappedCode,
		Input:        input.Input,
		TimeoutMS:    timeoutMS,
	})
	if err != nil {
		return nil, err
	}

	result, err := ParseSandboxCodeExecutionOutput(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}
