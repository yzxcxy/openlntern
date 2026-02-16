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
