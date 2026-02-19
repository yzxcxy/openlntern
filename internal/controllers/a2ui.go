package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/response"
	"openIntern/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateA2UI 创建 A2UI
func CreateA2UI(c *gin.Context) {
	var a2ui models.A2UI
	if err := c.ShouldBindJSON(&a2ui); err != nil {
		response.BadRequest(c)
		return
	}

	// 自定义 A2UI 必须关联当前用户，否则列表接口按 user_id 过滤会查不到
	userID := c.GetString("user_id")
	if userID != "" {
		a2ui.UserID = userID
	}
	if a2ui.Type == "" {
		a2ui.Type = models.A2UITypeCustom
	}

	if err := services.A2UI.CreateA2UI(&a2ui); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusCreated, a2ui)
}

// GetA2UI 获取 A2UI
func GetA2UI(c *gin.Context) {
	id := c.Param("id")
	a2ui, err := services.A2UI.GetA2UIByID(id)
	if err != nil {
		response.NotFound(c, "a2ui not found")
		return
	}
	response.JSONSuccess(c, http.StatusOK, a2ui)
}

// UpdateA2UI 更新 A2UI
func UpdateA2UI(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c)
		return
	}

	// 从 Header 获取操作者 ID
	operatorID := c.GetHeader("X-User-ID")

	if err := services.A2UI.UpdateA2UI(id, updates, operatorID); err != nil {
		if err.Error() == "permission denied: only admin can update official a2ui" ||
			err.Error() == "permission denied: authentication required for official a2ui" ||
			err.Error() == "permission denied: invalid user" {
			response.Forbidden(c)
			return
		}
		response.InternalError(c)
		return
	}

	response.JSONMessage(c, http.StatusOK, "a2ui updated successfully")
}

// DeleteA2UI 删除 A2UI
func DeleteA2UI(c *gin.Context) {
	id := c.Param("id")
	operatorID := c.GetHeader("X-User-ID")

	if err := services.A2UI.DeleteA2UI(id, operatorID); err != nil {
		if err.Error() == "permission denied: only admin can delete official a2ui" ||
			err.Error() == "permission denied: authentication required for official a2ui" ||
			err.Error() == "permission denied: invalid user" {
			response.Forbidden(c)
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "a2ui deleted successfully")
}

// ListOfficialA2UIs 获取官方 A2UI 列表
func ListOfficialA2UIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	a2uis, total, err := services.A2UI.ListOfficialA2UIs(page, pageSize, keyword)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  a2uis,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// ListCustomA2UIs 获取自定义 A2UI 列表
func ListCustomA2UIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	userID := c.Query("user_id")
	keyword := c.Query("keyword")

	if userID == "" {
		if value, ok := c.Get("user_id"); ok {
			if id, ok := value.(string); ok {
				userID = id
			}
		}
		if userID == "" {
			response.BadRequest(c)
			return
		}
	}

	a2uis, total, err := services.A2UI.ListCustomA2UIs(page, pageSize, userID, keyword)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  a2uis,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}
