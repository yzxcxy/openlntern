package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateA2UI 创建 A2UI
func CreateA2UI(c *gin.Context) {
	var a2ui models.A2UI
	if err := c.ShouldBindJSON(&a2ui); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.A2UI.CreateA2UI(&a2ui); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, a2ui)
}

// GetA2UI 获取 A2UI
func GetA2UI(c *gin.Context) {
	id := c.Param("id")
	a2ui, err := services.A2UI.GetA2UIByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "a2ui not found"})
		return
	}
	c.JSON(http.StatusOK, a2ui)
}

// UpdateA2UI 更新 A2UI
func UpdateA2UI(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从 Header 获取操作者 ID
	operatorID := c.GetHeader("X-User-ID")

	if err := services.A2UI.UpdateA2UI(id, updates, operatorID); err != nil {
		if err.Error() == "permission denied: only admin can update official a2ui" ||
			err.Error() == "permission denied: authentication required for official a2ui" ||
			err.Error() == "permission denied: invalid user" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "a2ui updated successfully"})
}

// DeleteA2UI 删除 A2UI
func DeleteA2UI(c *gin.Context) {
	id := c.Param("id")
	operatorID := c.GetHeader("X-User-ID")

	if err := services.A2UI.DeleteA2UI(id, operatorID); err != nil {
		if err.Error() == "permission denied: only admin can delete official a2ui" ||
			err.Error() == "permission denied: authentication required for official a2ui" ||
			err.Error() == "permission denied: invalid user" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "a2ui deleted successfully"})
}

// ListA2UIs 获取 A2UI 列表
func ListA2UIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	userIDStr := c.Query("user_id")

	var userID uint
	if userIDStr != "" {
		id, _ := strconv.Atoi(userIDStr)
		userID = uint(id)
	}

	a2uis, total, err := services.A2UI.ListA2UIs(page, pageSize, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  a2uis,
		"total": total,
		"page":  page,
	})
}

// ListOfficialA2UIs 获取官方 A2UI 列表
func ListOfficialA2UIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	a2uis, total, err := services.A2UI.ListOfficialA2UIs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  a2uis,
		"total": total,
		"page":  page,
	})
}

// ListCustomA2UIs 获取自定义 A2UI 列表
func ListCustomA2UIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	userIDStr := c.Query("user_id")

	var userID uint
	if userIDStr != "" {
		id, _ := strconv.Atoi(userIDStr)
		userID = uint(id)
	}

	a2uis, total, err := services.A2UI.ListCustomA2UIs(page, pageSize, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  a2uis,
		"total": total,
		"page":  page,
	})
}
