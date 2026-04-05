package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"openIntern/internal/response"
	chatsvc "openIntern/internal/services/chat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListThreads(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	threads, total, err := chatsvc.Thread.ListThreads(userID, page, pageSize)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  threads,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func GetThread(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	threadID := c.Param("thread_id")
	thread, err := chatsvc.Thread.GetThread(userID, threadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "thread not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, thread)
}

func UpdateThread(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	threadID := c.Param("thread_id")
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
		response.BadRequest(c)
		return
	}
	if err := chatsvc.Thread.UpdateThreadTitle(userID, threadID, req.Title); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "thread not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "thread updated successfully")
}

func DeleteThread(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	threadID := c.Param("thread_id")
	if err := chatsvc.Thread.DeleteThread(userID, threadID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "thread not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "thread deleted successfully")
}
