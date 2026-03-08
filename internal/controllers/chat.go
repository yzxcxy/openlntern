package controllers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"openIntern/internal/response"
	"openIntern/internal/services"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/gin-gonic/gin"
)

func ChatSSE(c *gin.Context) {
	var input types.RunAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("ChatSSE bind failed client_ip=%s err=%v", c.ClientIP(), err)
		response.BadRequest(c)
		return
	}

	if input.ThreadID == "" {
		input.ThreadID = events.GenerateThreadID()
	}
	ownerID := c.GetString("user_id")
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	if _, err := services.Thread.EnsureThread(input.ThreadID, ""); err != nil {
		log.Printf("ChatSSE ensure thread failed thread_id=%s client_ip=%s err=%v", input.ThreadID, c.ClientIP(), err)
		response.InternalError(c)
		return
	}

	log.Printf("ChatSSE start thread_id=%s run_id=%s messages=%d client_ip=%s", input.ThreadID, input.RunID, len(input.Messages), c.ClientIP())

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Stream(func(w io.Writer) bool {
		if err := services.RunAgent(c.Request.Context(), w, &input); err != nil {
			log.Printf("ChatSSE run failed thread_id=%s run_id=%s client_ip=%s err=%v", input.ThreadID, input.RunID, c.ClientIP(), err)
		} else {
			log.Printf("ChatSSE run success thread_id=%s run_id=%s client_ip=%s", input.ThreadID, input.RunID, c.ClientIP())
		}
		return false
	})
}

// UploadChatAsset handles chat attachment uploads and returns uploaded URL metadata.
func UploadChatAsset(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	threadID := strings.TrimSpace(c.PostForm("thread_id"))

	asset, err := services.ChatUpload.Upload(c.Request.Context(), ownerID, threadID, fileHeader)
	if err != nil {
		if errors.Is(err, services.ErrChatUploadValidation) {
			message := strings.TrimPrefix(err.Error(), services.ErrChatUploadValidation.Error()+": ")
			response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, strings.TrimSpace(message))
			return
		}
		log.Printf("UploadChatAsset failed owner_id=%s thread_id=%s file=%s err=%v", ownerID, threadID, fileHeader.Filename, err)
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, asset)
}
