package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"openIntern/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCodePluginToolInvokableRunUsesMainWrapperAndReturnsParsedJSON(t *testing.T) {
	tool, err := NewCodePluginTool(models.Tool{
		ToolName:        "echo_input",
		Description:     "echo",
		CodeLanguage:    "python",
		Code:            "def main(params: dict) -> dict:\n    return {\"result\": \"hello:\" + params.get(\"input\", \"\")}",
		BodyFieldsJSON:  `[{"name":"input","type":"string","required":true}]`,
		InputSchemaJSON: `{"type":"object","properties":{"input":{"type":"string"}}}`,
		TimeoutMS:       30000,
	})
	if err != nil {
		t.Fatalf("NewCodePluginTool returned error: %v", err)
	}

	codeTool, ok := tool.(*codePluginTool)
	if !ok {
		t.Fatalf("unexpected tool type: %T", tool)
	}
	codeTool.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			if r.URL.Path != sandboxCodeRunPath {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request failed: %v", err)
			}

			code, _ := payload["code"].(string)
			if !strings.Contains(code, "__openintern_result = main(__openintern_params)") {
				t.Fatalf("wrapped code did not call main: %s", code)
			}

			input, _ := payload["input"].(map[string]any)
			if got, _ := input["input"].(string); got != "A" {
				t.Fatalf("unexpected input payload: %#v", input)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"stdout":"{\"result\":\"hello:A\"}\n"}}`)),
			}, nil
		}),
	}

	ctx := context.WithValue(context.Background(), ContextKeySandboxBaseURL, "http://sandbox.local")
	result, err := codeTool.InvokableRun(ctx, `{"input":"A"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	if result != `{"result":"hello:A"}` {
		t.Fatalf("unexpected result: %s", result)
	}
}
