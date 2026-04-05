package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/response"
	a2uisvc "openIntern/internal/services/a2ui"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateA2UI 创建 A2UI
func CreateA2UI(c *gin.Context) {
	var a2ui models.A2UI
	if err := c.ShouldBindJSON(&a2ui); err != nil {
		response.BadRequest(c)
		return
	}
	a2ui.UserID = strings.TrimSpace(c.GetString("user_id"))

	if err := a2uisvc.A2UI.CreateA2UI(&a2ui); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusCreated, a2ui)
}

// GetA2UI 获取 A2UI
func GetA2UI(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	id := c.Param("id")
	a2ui, err := a2uisvc.A2UI.GetA2UIByID(userID, id)
	if err != nil {
		response.NotFound(c, "a2ui not found")
		return
	}
	response.JSONSuccess(c, http.StatusOK, a2ui)
}

// UpdateA2UI 更新 A2UI
func UpdateA2UI(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c)
		return
	}
	if err := a2uisvc.A2UI.UpdateA2UI(userID, id, updates); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONMessage(c, http.StatusOK, "a2ui updated successfully")
}

// DeleteA2UI 删除 A2UI
func DeleteA2UI(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	id := c.Param("id")
	if err := a2uisvc.A2UI.DeleteA2UI(userID, id); err != nil {
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "a2ui deleted successfully")
}

// ListA2UIs 获取 A2UI 列表
func ListA2UIs(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	a2uis, total, err := a2uisvc.A2UI.ListA2UIs(userID, page, pageSize, keyword)
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
