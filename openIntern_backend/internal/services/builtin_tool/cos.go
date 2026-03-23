package builtin_tool

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

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
	SandboxPath string `json:"sandbox_path" jsonschema_description:"沙箱内文件绝对路径，例如 /tmp/output.md"`
}

type sandboxReadRequest struct {
	File string `json:"file,omitempty"`
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
	// AIO Sandbox 文档示例统一使用绝对路径；这里强制约束，避免相对路径受工作目录影响。
	if !path.IsAbs(sandboxPath) {
		return "", errors.New("sandbox_path must be an absolute path in sandbox, for example /tmp/output.md")
	}
	decoded, err := readSandboxFile(ctx, baseURL, sandboxPath)
	if err != nil {
		log.Printf("upload_to_cos sandbox read failed cos_key=%s sandbox_path=%s err=%v", cosKey, sandboxPath, err)
		return "", errors.New("upload_to_cos failed at sandbox file read: " + err.Error())
	}
	url, err := uploader.UploadReader(ctx, cosKey, bytes.NewReader(decoded), "")
	if err != nil {
		log.Printf("upload_to_cos cos upload failed cos_key=%s sandbox_path=%s bytes=%d err=%v", cosKey, sandboxPath, len(decoded), err)
		return "", errors.New("upload_to_cos failed at cos upload: " + err.Error())
	}
	respPayload, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		return "", err
	}
	log.Printf("upload_to_cos success cos_key=%s sandbox_path=%s bytes=%d", cosKey, sandboxPath, len(decoded))
	return string(respPayload), nil
}

func readSandboxFile(ctx context.Context, baseURL string, sandboxPath string) ([]byte, error) {
	// 按 sandbox.agent-infra.com 文档要求读取文件：
	// POST /v1/file/read，入参字段为 file，响应正文位于 data.content。
	return readSandboxFileWithPayload(ctx, baseURL, sandboxReadRequest{
		File: sandboxPath,
	})
}

func readSandboxFileWithPayload(ctx context.Context, baseURL string, reqPayload sandboxReadRequest) ([]byte, error) {
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/file/read", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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
		decoded, err := base64.StdEncoding.DecodeString(readResp.Data.Content)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}
	return []byte(readResp.Data.Content), nil
}

func GetCOSTools(ctx context.Context) ([]einoTool.BaseTool, error) {
	uploadTool, err := utils.InferTool[UploadToCOSInput, string](
		"upload_to_cos",
		"从沙箱绝对路径读取文件并上传到 COS，入参为 cos_key 与 sandbox_path（必须是绝对路径，例如 /tmp/output.md），返回 url；失败时会明确指出是参数校验、沙箱读文件还是 COS 上传阶段出错。",
		uploadToCOSImpl,
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{uploadTool}, nil
}
