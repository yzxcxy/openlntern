package builtin_tool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	sandboxsvc "openIntern/internal/services/sandbox"
	storagesvc "openIntern/internal/services/storage"
	"path"
	"path/filepath"
	"strings"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type FileUploader interface {
	UploadReader(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
}

const (
	ContextKeyFileUploader contextKey = "openintern_file_uploader"
)

type UploadToObjectStorageInput struct {
	Purpose       string   `json:"purpose" jsonschema_description:"对象用途：chat/avatar/plugin"`
	ScopeSegments []string `json:"scope_segments" jsonschema_description:"可选的对象作用域分段，每项必须是单个路径段"`
	SandboxPath   string   `json:"sandbox_path" jsonschema_description:"沙箱内文件绝对路径，例如 /tmp/output.md"`
	ContentType   string   `json:"content_type,omitempty" jsonschema_description:"可选，文件 MIME 类型；例如 text/markdown"`
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

func uploadToObjectStorageImpl(ctx context.Context, input UploadToObjectStorageInput) (string, error) {
	userID, _ := ctx.Value(ContextKeyUserID).(string)
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", errors.New("user_id is required")
	}
	purpose, err := normalizeObjectPurpose(input.Purpose)
	if err != nil {
		return "", err
	}
	sandboxPath := strings.TrimSpace(input.SandboxPath)
	if sandboxPath == "" {
		return "", errors.New("sandbox_path is required")
	}
	// AIO Sandbox 文档示例统一使用绝对路径；这里强制约束，避免相对路径受工作目录影响。
	if !path.IsAbs(sandboxPath) {
		return "", errors.New("sandbox_path must be an absolute path in sandbox, for example /tmp/output.md")
	}
	instance, err := sandboxsvc.Lifecycle.GetOrCreate(ctx, userID)
	if err != nil {
		return "", err
	}
	decoded, err := readSandboxFile(ctx, instance.Endpoint, sandboxPath)
	if err != nil {
		log.Printf("upload_to_object_storage sandbox read failed user_id=%s purpose=%s sandbox_path=%s err=%v", userID, purpose, sandboxPath, err)
		return "", errors.New("upload_to_object_storage failed at sandbox file read: " + err.Error())
	}
	contentType := strings.TrimSpace(input.ContentType)
	uploaded, err := storagesvc.ObjectStorage.UploadUserObject(ctx, userID, storagesvc.UploadUserObjectSpec{
		Purpose:          purpose,
		ScopeSegments:    input.ScopeSegments,
		OriginalFileName: filepath.Base(sandboxPath),
		ContentType:      contentType,
	}, bytes.NewReader(decoded), int64(len(decoded)))
	if err != nil {
		log.Printf("upload_to_object_storage upload failed user_id=%s purpose=%s sandbox_path=%s bytes=%d err=%v", userID, purpose, sandboxPath, len(decoded), err)
		return "", errors.New("upload_to_object_storage failed at object storage upload: " + err.Error())
	}
	respPayload, err := json.Marshal(map[string]string{
		"key": uploaded.Key,
		"url": uploaded.URL,
	})
	if err != nil {
		return "", err
	}
	_ = sandboxsvc.Lifecycle.Touch(ctx, userID)
	log.Printf("upload_to_object_storage success user_id=%s purpose=%s key=%s sandbox_path=%s bytes=%d", userID, purpose, uploaded.Key, sandboxPath, len(decoded))
	return string(respPayload), nil
}

func normalizeObjectPurpose(value string) (storagesvc.ObjectPurpose, error) {
	// Restrict purposes to the backend's known namespaces to prevent arbitrary path categories.
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(storagesvc.ObjectPurposeChat):
		return storagesvc.ObjectPurposeChat, nil
	case string(storagesvc.ObjectPurposeAvatar):
		return storagesvc.ObjectPurposeAvatar, nil
	case string(storagesvc.ObjectPurposePlugin):
		return storagesvc.ObjectPurposePlugin, nil
	default:
		return "", errors.New("purpose must be one of: chat, avatar, plugin")
	}
}

func readSandboxFile(ctx context.Context, baseURL string, sandboxPath string) ([]byte, error) {
	// 按 sandbox.agent-infra.com 文档要求读取文件：
	// POST /v1/file/read，入参字段为 file，响应正文位于 data.content。
	return readSandboxFileWithPayload(ctx, baseURL, sandboxReadRequest{
		File: sandboxPath,
	})
}

func readSandboxFileWithPayload(ctx context.Context, baseURL string, reqPayload sandboxReadRequest) ([]byte, error) {
	respBody, err := sandboxsvc.Lifecycle.Client().ReadFile(ctx, baseURL, reqPayload.File)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func GetObjectStorageTools(ctx context.Context) ([]einoTool.BaseTool, error) {
	uploadTool, err := utils.InferTool[UploadToObjectStorageInput, string](
		"upload_to_object_storage",
		"从沙箱绝对路径读取文件并上传到对象存储，入参为 purpose、scope_segments、sandbox_path（必须是绝对路径）以及可选 content_type；返回 key 与 url，失败时会明确指出是参数校验、沙箱读文件还是对象存储上传阶段出错。",
		uploadToObjectStorageImpl,
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{uploadTool}, nil
}
