package dao

import (
	"context"
	"errors"
	"path"
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"
	"openIntern/internal/services/storage"
)

// KBLocalDAO 知识库本地存储DAO，管理数据库记录和MinIO文件。
type KBLocalDAO struct{}

var KBLocal = new(KBLocalDAO)

var (
	ErrKBNotFound      = errors.New("knowledge base not found")
	ErrKBAlreadyExists = errors.New("knowledge base already exists")
)

// Create 创建知识库记录和目录树条目。
func (d *KBLocalDAO) Create(ctx context.Context, kb *models.KnowledgeBase, entries []models.KBTreeEntry) error {
	if kb == nil {
		return errors.New("kb is nil")
	}
	// 检查是否已存在
	var existing models.KnowledgeBase
	err := database.DB.Where("user_id = ? AND name = ?", kb.UserID, kb.Name).First(&existing).Error
	if err == nil {
		return ErrKBAlreadyExists
	}
	// 创建KB记录
	if err := database.DB.Create(kb).Error; err != nil {
		return err
	}
	// 设置entries的KBID
	for i := range entries {
		entries[i].KBID = kb.ID
	}
	// 批量创建目录树条目
	if len(entries) > 0 {
		if err := database.DB.CreateInBatches(entries, 100).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetByName 根据用户ID和名称获取知识库。
func (d *KBLocalDAO) GetByName(ctx context.Context, userID, name string) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	err := database.DB.Where("user_id = ? AND name = ?", userID, name).First(&kb).Error
	if err != nil {
		return nil, ErrKBNotFound
	}
	return &kb, nil
}

// List 列出用户的所有知识库。
func (d *KBLocalDAO) List(ctx context.Context, userID string) ([]models.KnowledgeBase, error) {
	var kbs []models.KnowledgeBase
	err := database.DB.Where("user_id = ?", userID).Find(&kbs).Error
	if err != nil {
		return nil, err
	}
	return kbs, nil
}

// Delete 删除知识库及其目录树条目和MinIO文件。
func (d *KBLocalDAO) Delete(ctx context.Context, userID, name string) error {
	kb, err := d.GetByName(ctx, userID, name)
	if err != nil {
		return err
	}
	// 删除MinIO文件
	if kb.LocalPath != "" {
		if err := storage.File.DeleteByPrefix(ctx, kb.LocalPath); err != nil {
			// 记录错误但继续删除数据库记录
		}
	}
	// 删除目录树条目
	if err := database.DB.Where("kb_id = ?", kb.ID).Delete(&models.KBTreeEntry{}).Error; err != nil {
		return err
	}
	// 删除KB记录
	if err := database.DB.Delete(kb).Error; err != nil {
		return err
	}
	return nil
}

// GetTreeEntries 获取知识库的目录树条目。
func (d *KBLocalDAO) GetTreeEntries(ctx context.Context, kbID uint) ([]models.KBTreeEntry, error) {
	var entries []models.KBTreeEntry
	err := database.DB.Where("kb_id = ?", kbID).Order("path").Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// UploadFile 上传文件到MinIO。
func (d *KBLocalDAO) UploadFile(ctx context.Context, localPath string, filePath string) error {
	_, err := storage.File.UploadPath(ctx, localPath+"/"+filePath, filePath)
	return err
}

// ReadFile 从MinIO读取文件内容。
func (d *KBLocalDAO) ReadFile(ctx context.Context, objectKey string) (string, error) {
	result, err := storage.ObjectStorage.ReadObject(ctx, objectKey)
	if err != nil {
		return "", err
	}
	defer result.Reader.Close()
	var buf strings.Builder
	buf.Grow(int(result.Size))
	for {
		b := make([]byte, 4096)
		n, err := result.Reader.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	return buf.String(), nil
}

// BuildLocalPath 构建本地存储路径前缀。
func (d *KBLocalDAO) BuildLocalPath(userID, kbName string) string {
	return path.Join("users", userID, "kbs", kbName)
}

// UpdateIndexStatus 更新知识库索引状态。
func (d *KBLocalDAO) UpdateIndexStatus(ctx context.Context, kbID uint, taskID, status, errMsg string) error {
	return database.DB.Model(&models.KnowledgeBase{}).
		Where("id = ?", kbID).
		Updates(map[string]any{
			"index_task_id": taskID,
			"index_status":  status,
			"index_error":   errMsg,
		}).Error
}