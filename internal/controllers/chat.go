package controllers

import (
	"io"

	"openIntern/internal/response"
	"openIntern/internal/services"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/gin-gonic/gin"
)

func ChatSSE(c *gin.Context) {
	var input types.RunAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Stream(func(w io.Writer) bool {
		services.RunAgent(c.Request.Context(), w, &input)
		return false
	})
}
