package builtin_tool

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type FileUploader interface {
	UploadReader(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
}

const (
	ContextKeyFileUploader   contextKey = "openintern_file_uploader"
	ContextKeySandboxBaseURL contextKey = "openintern_sandbox_base_url"
)

type UploadToCOSInput struct {
	COSKey      string `json:"cos_key" jsonschema_description:"COS 对象路径（Object Key）"`
	SandboxPath string `json:"sandbox_path" jsonschema_description:"沙箱内文件路径"`
}

type sandboxReadRequest struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
}

type sandboxReadResponse struct {
	Data struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	} `json:"data"`
}

func uploadToCOSImpl(ctx context.Context, input UploadToCOSInput) (string, error) {
	uploader, _ := ctx.Value(ContextKeyFileUploader).(FileUploader)
	if uploader == nil {
		return "", errors.New("file uploader not available in context")
	}
	baseURL, _ := ctx.Value(ContextKeySandboxBaseURL).(string)
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", errors.New("sandbox base url not configured")
	}
	cosKey := strings.TrimSpace(input.COSKey)
	if cosKey == "" {
		return "", errors.New("cos_key is required")
	}
	sandboxPath := strings.TrimSpace(input.SandboxPath)
	if sandboxPath == "" {
		return "", errors.New("sandbox_path is required")
	}
	reqPayload := sandboxReadRequest{
		Path:     sandboxPath,
		Encoding: "base64",
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/file/read", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
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
	var readResp sandboxReadResponse
	if err := json.Unmarshal(respBody, &readResp); err != nil {
		return "", err
	}
	if readResp.Data.Content == "" {
		return "", errors.New("sandbox read returned empty content")
	}
	if readResp.Data.Encoding != "" && strings.ToLower(readResp.Data.Encoding) != "base64" {
		return "", errors.New("sandbox read returned unsupported encoding")
	}
	decoded, err := base64.StdEncoding.DecodeString(readResp.Data.Content)
	if err != nil {
		return "", err
	}
	url, err := uploader.UploadReader(ctx, cosKey, bytes.NewReader(decoded), "")
	if err != nil {
		return "", err
	}
	respPayload, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		return "", err
	}
	return string(respPayload), nil
}

func GetCOSTools(ctx context.Context) ([]einoTool.BaseTool, error) {
	uploadTool, err := utils.InferTool[UploadToCOSInput, string](
		"upload_to_cos",
		"从沙箱读取文件并上传到 COS，入参为 cos_key 与 sandbox_path，返回 url。",
		uploadToCOSImpl,
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{uploadTool}, nil
}
