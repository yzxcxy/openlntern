package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	toolsvc "openIntern/internal/services/tools"
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
	wrappedCode, err := wrapCodeForDebug(codeLanguage, code, input.Input)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	output, err := toolsvc.RunCodeInSandbox(ctx, &http.Client{
		Timeout: time.Duration(timeoutMS) * time.Millisecond,
	}, sandboxBaseURL, toolsvc.SandboxCodeRunInput{
		CodeLanguage: codeLanguage,
		Code:         wrappedCode,
		Input:        input.Input,
		TimeoutMS:    timeoutMS,
	})
	if err != nil {
		return nil, err
	}

	result, err := parseCodeDebugOutput(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func parseCodeDebugOutput(raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty output")
	}

	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return trimmed, nil
	}

	obj, ok := payload.(map[string]any)
	if !ok {
		return payload, nil
	}

	successValue, hasSuccess := obj["success"]
	if !hasSuccess {
		return payload, nil
	}
	success, ok := successValue.(bool)
	if !ok {
		return payload, nil
	}

	message, _ := obj["message"].(string)
	if !success {
		if strings.TrimSpace(message) == "" {
			message = "sandbox execute failed"
		}
		return nil, errors.New(message)
	}

	data, _ := obj["data"].(map[string]any)
	if data == nil {
		return nil, nil
	}

	if result, ok, err := extractSandboxExecutionResult(data); err != nil {
		return nil, err
	} else if ok {
		return result, nil
	}
	return nil, nil
}

func wrapCodeForDebug(codeLanguage string, code string, input map[string]any) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	inputLiteral := fmt.Sprintf("%q", string(inputJSON))

	switch codeLanguage {
	case codeLanguagePython:
		return strings.TrimSpace(code) + "\n\n" +
			"import json\n" +
			"__openintern_params = json.loads(" + inputLiteral + ")\n" +
			"__openintern_result = main(__openintern_params)\n" +
			"print(json.dumps(__openintern_result, ensure_ascii=False))\n", nil
	case codeLanguageJavaScript:
		return strings.TrimSpace(code) + "\n\n" +
			"const __openintern_params = JSON.parse(" + inputLiteral + ");\n" +
			"Promise.resolve(main(__openintern_params))\n" +
			"  .then((result) => {\n" +
			"    console.log(JSON.stringify(result ?? null));\n" +
			"  })\n" +
			"  .catch((error) => {\n" +
			"    console.error(error);\n" +
			"    if (typeof process !== \"undefined\") {\n" +
			"      process.exitCode = 1;\n" +
			"    }\n" +
			"  });\n", nil
	default:
		return "", errors.New("code_language must be python or javascript")
	}
}

func extractSandboxExecutionResult(data map[string]any) (any, bool, error) {
	if stdout, ok := data["stdout"].(string); ok {
		return parseSandboxExecutionValue(stdout)
	}

	outputs, ok := data["outputs"].([]any)
	if !ok || len(outputs) == 0 {
		return nil, false, nil
	}

	for index := len(outputs) - 1; index >= 0; index-- {
		if text, ok := outputs[index].(string); ok {
			if result, found, err := parseSandboxExecutionValue(text); err != nil || found {
				return result, found, err
			}
		}
	}

	return nil, false, nil
}

func parseSandboxExecutionValue(raw string) (any, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false, nil
	}

	lines := strings.Split(trimmed, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}
		var payload any
		if err := json.Unmarshal([]byte(line), &payload); err == nil {
			return payload, true, nil
		}
		return line, true, nil
	}

	return nil, false, nil
}
