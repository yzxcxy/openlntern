package controllers

import (
	"context"
	"io"
	"log"

	"openIntern/internal/response"
	"openIntern/internal/services"
	"openIntern/internal/services/tools"

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
	// 注入 user_id 供 agent 内 A2UI 等工具使用
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), tools.ContextKeyUserID, ownerID))
	if _, err := services.Thread.EnsureThread(ownerID, input.ThreadID, ""); err != nil {
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
