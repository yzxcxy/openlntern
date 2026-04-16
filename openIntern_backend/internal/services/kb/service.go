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

const missingImportedRootURIMessage = "openviking import returned no root_uri"

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
			IndexStatus: normalizeIndexStatus(kb.IndexStatus),
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
	importedRootURI := openVikingURI
	if result != nil {
		taskID = strings.TrimSpace(result.TaskID)
		if strings.TrimSpace(result.RootURI) != "" {
			importedRootURI = strings.TrimSpace(result.RootURI)
		}
	}

	indexStatus, indexError := s.resolveImportedIndexState(result, err)

	// 创建KB记录（包含索引状态）
	kb := &models.KnowledgeBase{
		UserID:        userID,
		Name:          kbName,
		OpenVikingURI: importedRootURI,
		LocalPath:     localPath,
		IndexTaskID:   taskID,
		IndexStatus:   indexStatus,
		IndexError:    indexError,
	}
	if err := dao.KBLocal.Create(ctx, kb, treeEntries); err != nil {
		return nil, err
	}

	async := indexStatus == "pending" || indexStatus == "processing"
	resultStatus := "accepted"
	if !async {
		resultStatus = indexStatus
	}
	return &AsyncResult{Name: kbName, TaskID: taskID, Status: resultStatus, Async: async}, nil
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

// ValidateChatSelections 确保对话态临时选择的知识库不包含索引失败项。
func (s *Service) ValidateChatSelections(ctx context.Context, kbNames []string) error {
	userID, err := dao.OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return err
	}
	for _, kbName := range normalizeKnowledgeBaseNames(kbNames) {
		kb, err := dao.KBLocal.GetByName(ctx, userID, kbName)
		if err != nil {
			return err
		}
		if normalizeIndexStatus(kb.IndexStatus) == "failed" {
			return fmt.Errorf("知识库 %s 索引失败，暂不支持在对话中选择问答", kb.Name)
		}
	}
	return nil
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
		Status:   normalizeIndexStatus(kb.IndexStatus),
		Error:    kb.IndexError,
		Resource: kb.OpenVikingURI,
	}, nil
}

// RefreshIndexStatus 刷新知识库索引状态（按知识库根URI探测OpenViking语义层是否可用）。
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

	currentStatus := normalizeIndexStatus(kb.IndexStatus)

	// 已完成直接返回；失败状态仅对明确不可恢复的错误停止探测。
	if currentStatus == "completed" || (currentStatus == "failed" && !shouldProbeFailedKB(kb.IndexError)) {
		return &IndexStatusResult{
			TaskID:   kb.IndexTaskID,
			Status:   currentStatus,
			Error:    kb.IndexError,
			Resource: kb.OpenVikingURI,
		}, nil
	}

	newStatus, newError, err := s.probeIndexStatusByRootURI(ctx, kb)
	if err != nil {
		return &IndexStatusResult{
			TaskID:   kb.IndexTaskID,
			Status:   currentStatus,
			Error:    kb.IndexError,
			Resource: kb.OpenVikingURI,
		}, nil
	}
	if err := dao.KBLocal.UpdateIndexStatus(ctx, kb.ID, kb.IndexTaskID, newStatus, newError); err != nil {
		return &IndexStatusResult{
			TaskID:   kb.IndexTaskID,
			Status:   newStatus,
			Error:    newError,
			Resource: kb.OpenVikingURI,
		}, nil
	}

	return &IndexStatusResult{
		TaskID:   kb.IndexTaskID,
		Status:   newStatus,
		Error:    newError,
		Resource: kb.OpenVikingURI,
	}, nil
}

// resolveImportedIndexState determines the persisted KB index state after one import submission.
func (s *Service) resolveImportedIndexState(result *dao.ImportResult, importErr error) (string, string) {
	switch {
	case importErr != nil:
		return "failed", importErr.Error()
	case result == nil:
		return "failed", missingImportedRootURIMessage
	case strings.TrimSpace(result.RootURI) == "":
		return "failed", missingImportedRootURIMessage
	case normalizeIndexStatus(result.Status) == "failed":
		return "failed", strings.TrimSpace(result.Status)
	}
	return "pending", ""
}

func (s *Service) probeIndexStatusByRootURI(ctx context.Context, kb *models.KnowledgeBase) (string, string, error) {
	targetURI := strings.TrimSpace(kb.OpenVikingURI)
	if targetURI == "" {
		return "failed", missingImportedRootURIMessage, nil
	}
	if overview, err := dao.KnowledgeBase.ReadOverview(ctx, targetURI); err == nil {
		if strings.TrimSpace(overview) != "" {
			return "completed", "", nil
		}
	} else if !isProbePendingError(err) {
		return "", "", err
	}
	if abstract, err := dao.KnowledgeBase.ReadAbstract(ctx, targetURI); err == nil {
		if strings.TrimSpace(abstract) != "" {
			return "completed", "", nil
		}
	} else if !isProbePendingError(err) {
		return "", "", err
	}
	return "pending", "", nil
}

func isProbePendingError(err error) bool {
	if err == nil {
		return false
	}
	return IsNotFound(err)
}

func shouldProbeFailedKB(errMsg string) bool {
	lower := strings.ToLower(strings.TrimSpace(errMsg))
	if lower == "" {
		return true
	}
	return strings.Contains(lower, "no task_id")
}

// normalizeIndexStatus maps OpenViking and legacy status strings into the frontend contract.
func normalizeIndexStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "pending", "queued", "accepted", "created":
		return "pending"
	case "processing", "running", "in_progress", "in-progress":
		return "processing"
	case "completed", "success", "succeeded", "done", "finished":
		return "completed"
	case "failed", "error", "errored", "cancelled", "canceled", "timeout", "timed_out", "timed-out":
		return "failed"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}
