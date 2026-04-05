package controllers

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"syscall"

	"openIntern/internal/response"
	agentsvc "openIntern/internal/services/agent"
	chatsvc "openIntern/internal/services/chat"

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
	if _, err := chatsvc.Thread.EnsureThread(ownerID, input.ThreadID, ""); err != nil {
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
		runtimeCtx := agentsvc.WithOwnerID(c.Request.Context(), ownerID)
		if err := agentsvc.RunAgent(runtimeCtx, w, &input); err != nil {
			if isBenignSSECloseError(err) {
				log.Printf("ChatSSE client disconnected thread_id=%s run_id=%s client_ip=%s err=%v", input.ThreadID, input.RunID, c.ClientIP(), err)
			} else {
				log.Printf("ChatSSE run failed thread_id=%s run_id=%s client_ip=%s err=%v", input.ThreadID, input.RunID, c.ClientIP(), err)
			}
		} else {
			log.Printf("ChatSSE run success thread_id=%s run_id=%s client_ip=%s", input.ThreadID, input.RunID, c.ClientIP())
		}
		return false
	})
}

// isBenignSSECloseError identifies client-side stream termination that should not be treated as a server failure.
func isBenignSSECloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "broken pipe") || strings.Contains(lower, "reset by peer")
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

	asset, err := chatsvc.ChatUpload.Upload(c.Request.Context(), ownerID, threadID, fileHeader)
	if err != nil {
		if errors.Is(err, chatsvc.ErrChatUploadValidation) {
			message := strings.TrimPrefix(err.Error(), chatsvc.ErrChatUploadValidation.Error()+": ")
			response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, strings.TrimSpace(message))
			return
		}
		log.Printf("UploadChatAsset failed owner_id=%s thread_id=%s file=%s err=%v", ownerID, threadID, fileHeader.Filename, err)
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, asset)
}
