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

type kbMovePayload struct {
	FromURI string `json:"from_uri"`
	ToURI   string `json:"to_uri"`
}

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
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusAccepted, result)
}

func UploadKnowledgeBaseFile(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	result, err := kbsvc.KnowledgeBase.UploadFile(
		ctx,
		c.PostForm("kb_name"),
		c.PostForm("target"),
		fileHeader,
	)
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusAccepted, result)
}

func MoveKnowledgeBaseEntry(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	var payload kbMovePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c)
		return
	}
	result, err := kbsvc.KnowledgeBase.MoveEntry(ctx, payload.FromURI, payload.ToURI)
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, result)
}

func DragKnowledgeBaseEntry(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	var payload kbMovePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c)
		return
	}
	result, err := kbsvc.KnowledgeBase.DragEntry(ctx, payload.FromURI, payload.ToURI)
	if err != nil {
		switch {
		case errors.Is(err, kbsvc.ErrNotConfigured):
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		case errors.Is(err, kbsvc.ErrInvalidInput):
			response.BadRequest(c)
		default:
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		}
		return
	}
	response.JSONSuccess(c, http.StatusOK, result)
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

func DeleteKnowledgeBaseEntry(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	uri, err := kbsvc.KnowledgeBase.DeleteEntry(
		ctx,
		c.Query("uri"),
		strings.EqualFold(c.DefaultQuery("recursive", "false"), "true"),
	)
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
	response.JSONSuccess(c, http.StatusOK, gin.H{"uri": uri})
}

func GetKnowledgeBaseContent(c *gin.Context) {
	ctx, ok := knowledgeBaseRequestContext(c)
	if !ok {
		return
	}
	content, err := kbsvc.KnowledgeBase.ReadContent(ctx, c.Query("uri"))
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
