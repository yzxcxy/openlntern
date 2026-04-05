package sandbox

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	codeExecutePath = "/v1/code/execute"
	fileReadPath    = "/v1/file/read"
)

// CodeRunInput 描述一次代码执行请求。
type CodeRunInput struct {
	CodeLanguage string
	Code         string
	Input        any
	TimeoutMS    int
}

type sandboxReadResponse struct {
	Success *bool  `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		File     string `json:"file"`
	} `json:"data"`
}

// Client 封装对用户 sandbox 实例的 HTTP / MCP 调用。
type Client struct {
	httpClient *http.Client
}

// NewClient 创建一个默认超时的 sandbox client。
func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) ExecuteCode(ctx context.Context, endpoint string, input CodeRunInput) (string, error) {
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

	respBody, err := c.doJSONRequest(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+codeExecutePath, body)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(respBody))
	if result == "" {
		return "", errors.New("sandbox execute returned empty response")
	}
	return result, nil
}

func (c *Client) ReadFile(ctx context.Context, endpoint, sandboxPath string) ([]byte, error) {
	body, err := json.Marshal(map[string]string{"file": sandboxPath})
	if err != nil {
		return nil, err
	}

	respBody, err := c.doJSONRequest(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+fileReadPath, body)
	if err != nil {
		return nil, err
	}

	var readResp sandboxReadResponse
	if err := json.Unmarshal(respBody, &readResp); err != nil {
		return nil, err
	}
	if readResp.Success != nil && !*readResp.Success {
		msg := strings.TrimSpace(readResp.Message)
		if msg == "" {
			msg = "sandbox read failed"
		}
		return nil, errors.New(msg)
	}
	if readResp.Data.Content == "" {
		msg := strings.TrimSpace(readResp.Message)
		if msg == "" {
			msg = "sandbox read returned empty content"
		}
		return nil, errors.New(msg)
	}
	if readResp.Data.Encoding != "" && strings.ToLower(readResp.Data.Encoding) != "base64" {
		return nil, errors.New("sandbox read returned unsupported encoding")
	}
	if strings.ToLower(readResp.Data.Encoding) == "base64" {
		return base64.StdEncoding.DecodeString(readResp.Data.Content)
	}
	return []byte(readResp.Data.Content), nil
}

func (c *Client) ExecuteBash(ctx context.Context, endpoint, command string) (string, error) {
	mcpURL := strings.TrimRight(endpoint, "/") + "/mcp"
	cli, err := client.NewStreamableHttpClient(mcpURL)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = cli.Close()
	}()

	if err := cli.Start(ctx); err != nil {
		return "", err
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "openintern",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		return "", err
	}

	result, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sandbox_execute_bash",
			Arguments: map[string]any{"command": command},
		},
	})
	if err != nil {
		return "", err
	}
	if result.IsError {
		payload, _ := json.Marshal(result)
		return "", errors.New(strings.TrimSpace(string(payload)))
	}
	if result.StructuredContent != nil {
		payload, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	}
	payload, err := json.Marshal(result.Content)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (c *Client) doJSONRequest(ctx context.Context, method, requestURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return respBody, nil
}
