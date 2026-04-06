package controllers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"openIntern/internal/dao"
	"openIntern/internal/response"
	kbsvc "openIntern/internal/services/kb"

	"github.com/gin-gonic/gin"
)

func ListKnowledgeBases(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	items, err := kbsvc.KnowledgeBase.List(ctx)
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case kbsvc.IsNotFound(err):
			response.JSONSuccess(c, http.StatusOK, []kbsvc.Item{})
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func GetKnowledgeBaseTree(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	entries, err := kbsvc.KnowledgeBase.Tree(ctx, c.Param("name"))
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		case kbsvc.IsNotFound(err):
			response.JSONSuccess(c, http.StatusOK, []map[string]any{})
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, entries)
}

func ImportKnowledgeBase(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil && !isMissingKnowledgeBaseImportFile(err) {
		response.BadRequest(c)
		return
	}
	result, err := kbsvc.KnowledgeBase.Import(ctx, c.PostForm("kb_name"), fileHeader)
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		case errors.Is(err, kbsvc.ErrInvalidZipPath):
			response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		case errors.Is(err, kbsvc.ErrKBExists):
			response.JSONError(c, http.StatusConflict, response.CodeBadRequest, "知识库已存在，请先删除后再导入")
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusAccepted, result)
}

func DeleteKnowledgeBase(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	kbName, err := kbsvc.KnowledgeBase.Delete(ctx, c.Param("name"))
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		case kbsvc.IsNotFound(err):
			response.NotFound(c, "kb not found")
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"name": kbName})
}

func GetKnowledgeBaseContent(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	kbName := strings.TrimSpace(c.Query("kb_name"))
	path := strings.TrimSpace(c.Query("path"))
	if kbName == "" || path == "" {
		response.BadRequest(c)
		return
	}
	content, err := kbsvc.KnowledgeBase.ReadContent(ctx, kbName, path)
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		case kbsvc.IsNotFound(err):
			response.NotFound(c, "entry not found")
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"content": content})
}

// GetImportTaskStatus returns the status of an async import task.
func GetImportTaskStatus(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		response.BadRequest(c)
		return
	}
	result, err := dao.OpenVikingSession.GetTask(ctx, taskID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			response.NotFound(c, "task not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"task_id":     result.TaskID,
		"status":      result.Status,
		"resource_id": result.ResourceID,
		"error":       result.Error,
	})
}

func isMissingKnowledgeBaseImportFile(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, http.ErrMissingFile) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "missing file")
}

func knowledgeBaseRequestContext(c *gin.Context) (context.Context, bool) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	if userID == "" {
		response.Unauthorized(c)
		return nil, false
	}
	return dao.WithOpenVikingUserID(c.Request.Context(), userID), true
}

// GetKBIndexStatus 获取知识库索引状态。
func GetKBIndexStatus(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	result, err := kbsvc.KnowledgeBase.GetIndexStatus(ctx, c.Param("name"))
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		case kbsvc.IsNotFound(err):
			response.NotFound(c, "kb not found")
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, result)
}

// RefreshKBIndexStatus 刷新知识库索引状态（查询OpenViking并更新）。
func RefreshKBIndexStatus(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	result, err := kbsvc.KnowledgeBase.RefreshIndexStatus(ctx, c.Param("name"))
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		case kbsvc.IsNotFound(err):
			response.NotFound(c, "kb not found")
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, result)
}