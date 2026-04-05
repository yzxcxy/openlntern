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

func ListMessages(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	threadID := c.Param("thread_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	messages, total, err := chatsvc.Message.ListMessages(userID, threadID, page, pageSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "thread not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  messages,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}
