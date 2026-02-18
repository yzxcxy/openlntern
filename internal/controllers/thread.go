package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"openIntern/internal/response"
	"openIntern/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListThreads(c *gin.Context) {
	ownerID := c.GetString("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	threads, total, err := services.Thread.ListThreads(ownerID, page, pageSize)
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
	ownerID := c.GetString("user_id")
	threadID := c.Param("thread_id")
	thread, err := services.Thread.GetThread(ownerID, threadID)
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
	ownerID := c.GetString("user_id")
	threadID := c.Param("thread_id")
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
		response.BadRequest(c)
		return
	}
	if err := services.Thread.UpdateThreadTitle(ownerID, threadID, req.Title); err != nil {
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
	ownerID := c.GetString("user_id")
	threadID := c.Param("thread_id")
	if err := services.Thread.DeleteThread(ownerID, threadID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "thread not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "thread deleted successfully")
}
