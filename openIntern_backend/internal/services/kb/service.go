package kb

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/models"
	"openIntern/internal/services/storage"
)

var (
	ErrNotConfigured  = errors.New("knowledge base storage not configured")
	ErrInvalidInput   = errors.New("invalid knowledge base input")
	ErrInvalidZipPath = errors.New("invalid zip entry path")
	ErrKBExists       = errors.New("knowledge base already exists")
)

type Item struct {
	Name        string `json:"name"`
	IndexStatus string `json:"index_status,omitempty"`
}

type TreeEntry struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

type AsyncResult struct {
	Name   string `json:"name,omitempty"`
	TaskID string `json:"task_id,omitempty"`
	Status string `json:"status"`
	Async  bool   `json:"async"`
}

type IndexStatusResult struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Resource string `json:"resource_id,omitempty"`
}

type Service struct{}

var KnowledgeBase = new(Service)

func (s *Service) Configured() bool {
	return dao.KnowledgeBase.Configured()
}

func (s *Service) ensureConfigured() error {
	if !s.Configured() {
		return ErrNotConfigured
	}
	return nil
}

// List 列出用户的所有知识库。
func (s *Service) List(ctx context.Context) ([]Item, error) {
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	kbs, err := dao.KBLocal.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Item, 0, len(kbs))
	for _, kb := range kbs {
		result = append(result, Item{
			Name:        kb.Name,
			IndexStatus: kb.IndexStatus,
		})
	}
	return result, nil
}

// Tree 获取知识库的目录树。
func (s *Service) Tree(ctx context.Context, rawName string) ([]TreeEntry, error) {
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	kb, err := dao.KBLocal.GetByName(ctx, userID, kbName)
	if err != nil {
		return nil, err
	}
	entries, err := dao.KBLocal.GetTreeEntries(ctx, kb.ID)
	if err != nil {
		return nil, err
	}
	result := make([]TreeEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, TreeEntry{
			Path:  e.Path,
			Name:  e.Name,
			IsDir: e.IsDir,
			Size:  e.Size,
		})
	}
	return result, nil
}

// Import 导入知识库（双存储：本地MinIO + OpenViking）。
func (s *Service) Import(ctx context.Context, rawName string, fileHeader *multipart.FileHeader) (*AsyncResult, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	if fileHeader == nil {
		return nil, fmt.Errorf("%w: file is required", ErrInvalidInput)
	}

	// 检查是否已存在
	if _, err := dao.KBLocal.GetByName(ctx, userID, kbName); err == nil {
		return nil, ErrKBExists
	}

	// 保存上传的zip到临时文件
	tempDir, err := os.MkdirTemp("", "kb-import-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	tempZipPath := filepath.Join(tempDir, "upload.zip")
	if err := saveMultipartFile(fileHeader, tempZipPath); err != nil {
		return nil, err
	}

	// 验证zip内容
	if err := validateZipArchive(tempZipPath); err != nil {
		return nil, err
	}

	// 解压zip到临时目录
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, err
	}
	treeEntries, err := ExtractZipTree(tempZipPath, extractDir)
	if err != nil {
		return nil, err
	}

	// 构建本地存储路径前缀
	localPath := dao.KBLocal.BuildLocalPath(userID, kbName)

	// 上传解压后的文件到MinIO
	if err := uploadExtractedFiles(ctx, extractDir, localPath); err != nil {
		return nil, err
	}

	// 构建OpenViking URI
	openVikingURI, err := dao.KnowledgeBase.URI(ctx, kbName)
	if err != nil {
		return nil, err
	}

	// 上传zip到OpenViking进行检索索引
	result, err := dao.KnowledgeBase.IngestZip(ctx, tempZipPath, openVikingURI, false, 0)
	taskID := ""
	if result != nil {
		taskID = result.TaskID
	}

	// 创建KB记录（包含索引状态）
	kb := &models.KnowledgeBase{
		UserID:        userID,
		Name:          kbName,
		OpenVikingURI: openVikingURI,
		LocalPath:     localPath,
		IndexTaskID:   taskID,
		IndexStatus:   "pending",
	}
	if taskID == "" {
		// OpenViking上传失败，标记状态为failed
		if err != nil {
			kb.IndexStatus = "failed"
			kb.IndexError = err.Error()
		}
	}
	if err := dao.KBLocal.Create(ctx, kb, treeEntries); err != nil {
		return nil, err
	}

	return &AsyncResult{Name: kbName, TaskID: taskID, Status: "accepted", Async: true}, nil
}

// uploadExtractedFiles 上传解压后的文件到MinIO。
func uploadExtractedFiles(ctx context.Context, extractDir string, localPath string) error {
	return filepath.Walk(extractDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// 计算相对路径
		relPath, err := filepath.Rel(extractDir, filePath)
		if err != nil {
			return err
		}
		// 构建对象键
		objectKey := localPath + "/" + filepath.ToSlash(relPath)
		// 上传文件
		_, err = storage.File.UploadPath(ctx, objectKey, filePath)
		return err
	})
}

// Delete 删除知识库（本地MinIO + OpenViking + 数据库）。
func (s *Service) Delete(ctx context.Context, rawName string) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return "", err
	}

	// 获取KB记录以获取OpenVikingURI
	kb, err := dao.KBLocal.GetByName(ctx, userID, kbName)
	if err != nil {
		return "", err
	}

	// 删除OpenViking中的资源
	if kb.OpenVikingURI != "" {
		if err := dao.KnowledgeBase.Delete(ctx, kbName); err != nil {
			// 记录错误但继续删除本地数据
		}
	}

	// 删除本地存储和数据库记录
	if err := dao.KBLocal.Delete(ctx, userID, kbName); err != nil {
		return "", err
	}
	return kbName, nil
}

// ReadContent 读取文件内容（从本地MinIO）。
func (s *Service) ReadContent(ctx context.Context, kbName string, filePath string) (string, error) {
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	kb, err := dao.KBLocal.GetByName(ctx, userID, kbName)
	if err != nil {
		return "", err
	}
	// 构建对象键
	objectKey := kb.LocalPath + "/" + filePath
	return dao.KBLocal.ReadFile(ctx, objectKey)
}

// ReadContentByURI 根据URI读取文件内容（从OpenViking，用于Agent检索）。
func (s *Service) ReadContentByURI(ctx context.Context, uri string) (string, error) {
	return dao.KnowledgeBase.ReadContent(ctx, uri)
}

func normalizeKnowledgeBaseName(rawName string) (string, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("%w: invalid name", ErrInvalidInput)
	}
	return name, nil
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, dao.ErrKBNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

// GetIndexStatus 获取知识库索引状态（不查询OpenViking）。
func (s *Service) GetIndexStatus(ctx context.Context, rawName string) (*IndexStatusResult, error) {
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	kb, err := dao.KBLocal.GetByName(ctx, userID, kbName)
	if err != nil {
		return nil, err
	}
	return &IndexStatusResult{
		TaskID:   kb.IndexTaskID,
		Status:   kb.IndexStatus,
		Error:    kb.IndexError,
		Resource: kb.OpenVikingURI,
	}, nil
}

// RefreshIndexStatus 刷新知识库索引状态（查询OpenViking并更新数据库）。
func (s *Service) RefreshIndexStatus(ctx context.Context, rawName string) (*IndexStatusResult, error) {
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	kb, err := dao.KBLocal.GetByName(ctx, userID, kbName)
	if err != nil {
		return nil, err
	}

	// 如果没有task_id或状态已完成/失败，直接返回当前状态
	if kb.IndexTaskID == "" || kb.IndexStatus == "completed" || kb.IndexStatus == "failed" {
		return &IndexStatusResult{
			TaskID:   kb.IndexTaskID,
			Status:   kb.IndexStatus,
			Error:    kb.IndexError,
			Resource: kb.OpenVikingURI,
		}, nil
	}

	// 查询OpenViking任务状态
	taskResult, err := dao.OpenVikingSession.GetTask(ctx, kb.IndexTaskID)
	if err != nil {
		// 查询失败，返回当前状态但不更新
		return &IndexStatusResult{
			TaskID:   kb.IndexTaskID,
			Status:   kb.IndexStatus,
			Error:    kb.IndexError,
			Resource: kb.OpenVikingURI,
		}, nil
	}

	// 更新数据库记录
	newStatus := taskResult.Status
	newError := ""
	if taskResult.Error != "" {
		newError = taskResult.Error
	}
	if err := dao.KBLocal.UpdateIndexStatus(ctx, kb.ID, kb.IndexTaskID, newStatus, newError); err != nil {
		// 更新失败，返回查询到的状态但不持久化
		return &IndexStatusResult{
			TaskID:   kb.IndexTaskID,
			Status:   newStatus,
			Error:    newError,
			Resource: taskResult.ResourceID,
		}, nil
	}

	return &IndexStatusResult{
		TaskID:   kb.IndexTaskID,
		Status:   newStatus,
		Error:    newError,
		Resource: taskResult.ResourceID,
	}, nil
}