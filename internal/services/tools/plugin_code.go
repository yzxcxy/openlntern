package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"openIntern/internal/models"
	"strings"
	"time"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einoJSONSchema "github.com/eino-contrib/jsonschema"
)

const sandboxCodeRunPath = "/v1/code/execute"

type SandboxCodeRunInput struct {
	CodeLanguage string
	Code         string
	Input        any
	TimeoutMS    int
}

type codePluginTool struct {
	def            models.Tool
	info           *schema.ToolInfo
	client         *http.Client
	inputFieldDefs []codePluginField
}

type codePluginField struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Required   bool              `json:"required"`
	EnumValues []string          `json:"enum_values"`
	Children   []codePluginField `json:"children"`
	Items      *codePluginField  `json:"items"`
}

var _ einoTool.InvokableTool = (*codePluginTool)(nil)

func NewCodePluginTool(def models.Tool) (einoTool.BaseTool, error) {
	info, err := buildCodePluginToolInfo(def)
	if err != nil {
		return nil, err
	}

	fieldDefs, err := decodeCodePluginFields(def.BodyFieldsJSON)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(def.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &codePluginTool{
		def:            def,
		info:           info,
		client:         &http.Client{Timeout: timeout},
		inputFieldDefs: fieldDefs,
	}, nil
}

func (t *codePluginTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *codePluginTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einoTool.Option) (string, error) {
	_ = opts

	baseURL, _ := ctx.Value(ContextKeySandboxBaseURL).(string)

	input, err := decodeCodePluginInput(argumentsInJSON)
	if err != nil {
		return "", err
	}
	if err := validateCodePluginObject(input, t.inputFieldDefs, ""); err != nil {
		return "", err
	}

	wrappedCode, err := WrapCodeForMainExecution(t.def.CodeLanguage, t.def.Code, input)
	if err != nil {
		return "", err
	}

	output, err := RunCodeInSandbox(ctx, t.client, baseURL, SandboxCodeRunInput{
		CodeLanguage: t.def.CodeLanguage,
		Code:         wrappedCode,
		Input:        input,
		TimeoutMS:    t.def.TimeoutMS,
	})
	if err != nil {
		return "", err
	}

	result, err := ParseSandboxCodeExecutionOutput(output)
	if err != nil {
		return "", err
	}

	return encodeCodePluginResult(result)
}

func RunCodeInSandbox(ctx context.Context, client *http.Client, baseURL string, input SandboxCodeRunInput) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", errors.New("sandbox base url not configured")
	}

	timeout := time.Duration(input.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	payload := map[string]any{
		"language":   input.CodeLanguage,
		"code":       input.Code,
		"input":      input.Input,
		"timeout_ms": input.TimeoutMS,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+sandboxCodeRunPath, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return "", errors.New(msg)
	}

	result := strings.TrimSpace(string(respBody))
	if result == "" {
		return "", errors.New("sandbox execute returned empty response")
	}
	return result, nil
}

func WrapCodeForMainExecution(codeLanguage string, code string, input map[string]any) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	inputLiteral := fmt.Sprintf("%q", string(inputJSON))

	switch strings.ToLower(strings.TrimSpace(codeLanguage)) {
	case "python":
		return strings.TrimSpace(code) + "\n\n" +
			"import json\n" +
			"__openintern_params = json.loads(" + inputLiteral + ")\n" +
			"__openintern_result = main(__openintern_params)\n" +
			"print(json.dumps(__openintern_result, ensure_ascii=False))\n", nil
	case "javascript":
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

func ParseSandboxCodeExecutionOutput(raw string) (any, error) {
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

	if result, ok, err := extractSandboxCodeExecutionResult(data); err != nil {
		return nil, err
	} else if ok {
		return result, nil
	}
	return nil, nil
}

func extractSandboxCodeExecutionResult(data map[string]any) (any, bool, error) {
	if stdout, ok := data["stdout"].(string); ok {
		return parseSandboxCodeExecutionValue(stdout)
	}

	outputs, ok := data["outputs"].([]any)
	if !ok || len(outputs) == 0 {
		return nil, false, nil
	}

	for index := len(outputs) - 1; index >= 0; index-- {
		if text, ok := outputs[index].(string); ok {
			if result, found, err := parseSandboxCodeExecutionValue(text); err != nil || found {
				return result, found, err
			}
		}
	}

	return nil, false, nil
}

func parseSandboxCodeExecutionValue(raw string) (any, bool, error) {
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

func encodeCodePluginResult(result any) (string, error) {
	if result == nil {
		return "null", nil
	}
	if text, ok := result.(string); ok {
		return text, nil
	}

	body, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func buildCodePluginToolInfo(def models.Tool) (*schema.ToolInfo, error) {
	inputSchema := strings.TrimSpace(def.InputSchemaJSON)
	if inputSchema == "" {
		return nil, errors.New("input_schema_json is required")
	}

	var js einoJSONSchema.Schema
	if err := json.Unmarshal([]byte(inputSchema), &js); err != nil {
		return nil, errors.New("input_schema_json must be valid json schema")
	}

	return &schema.ToolInfo{
		Name:        def.ToolName,
		Desc:        def.Description,
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&js),
	}, nil
}

func decodeCodePluginFields(raw string) ([]codePluginField, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []codePluginField{}, nil
	}

	var fields []codePluginField
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return nil, errors.New("body_fields_json must be valid json")
	}
	return fields, nil
}

func decodeCodePluginInput(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]any{}, nil
	}

	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()

	var payload any
	if err := dec.Decode(&payload); err != nil {
		return nil, errors.New("tool arguments must be valid json")
	}
	if payload == nil {
		return map[string]any{}, nil
	}

	obj, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("tool arguments must be a json object")
	}
	return obj, nil
}

func validateCodePluginObject(input map[string]any, fields []codePluginField, path string) error {
	allowed := make(map[string]codePluginField, len(fields))
	for _, field := range fields {
		allowed[field.Name] = field
	}

	for _, field := range fields {
		if field.Required {
			if _, ok := input[field.Name]; !ok {
				return fmt.Errorf("%s is required", joinCodePluginPath(path, field.Name))
			}
		}
	}

	for key, value := range input {
		field, ok := allowed[key]
		if !ok {
			return fmt.Errorf("%s is not allowed", joinCodePluginPath(path, key))
		}
		if err := validateCodePluginValue(value, field, joinCodePluginPath(path, key)); err != nil {
			return err
		}
	}

	return nil
}

func validateCodePluginValue(value any, field codePluginField, path string) error {
	if value == nil {
		return fmt.Errorf("%s must be %s", path, field.Type)
	}

	switch field.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be string", path)
		}
		if len(field.EnumValues) > 0 {
			for _, enumValue := range field.EnumValues {
				if str == enumValue {
					return nil
				}
			}
			return fmt.Errorf("%s must be one of %s", path, strings.Join(field.EnumValues, ", "))
		}
	case "number":
		if !isCodePluginNumber(value) {
			return fmt.Errorf("%s must be number", path)
		}
	case "integer":
		if !isCodePluginInteger(value) {
			return fmt.Errorf("%s must be integer", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be boolean", path)
		}
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be object", path)
		}
		if err := validateCodePluginObject(obj, field.Children, path); err != nil {
			return err
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be array", path)
		}
		if field.Items == nil {
			return fmt.Errorf("%s array items are not configured", path)
		}
		for idx, item := range arr {
			if err := validateCodePluginValue(item, *field.Items, fmt.Sprintf("%s[%d]", path, idx)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%s has unsupported type %s", path, field.Type)
	}

	return nil
}

func isCodePluginNumber(value any) bool {
	switch n := value.(type) {
	case json.Number:
		_, err := n.Float64()
		return err == nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isCodePluginInteger(value any) bool {
	switch n := value.(type) {
	case json.Number:
		if _, err := n.Int64(); err == nil {
			return true
		}
		fv, err := n.Float64()
		if err != nil {
			return false
		}
		return math.Trunc(fv) == fv
	case float64:
		return math.Trunc(n) == n
	case float32:
		return math.Trunc(float64(n)) == float64(n)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func joinCodePluginPath(parent string, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
