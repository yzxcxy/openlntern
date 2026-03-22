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

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	"github.com/gin-gonic/gin"
)

// DebugAgentSSEInfo documents debug SSE usage and avoids noisy GET 404 probes.
func DebugAgentSSEInfo(c *gin.Context) {
	response.JSONMessage(c, http.StatusOK, "debug sse endpoint is available, use POST with json body")
}

// DebugAgentSSE runs a transient agent definition for create/edit page testing without persistence.
func DebugAgentSSE(c *gin.Context) {
	var input types.RunAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Stream(func(w io.Writer) bool {
		runtimeCtx := agentsvc.WithOwnerID(c.Request.Context(), ownerID)
		if err := agentsvc.RunDebugAgent(runtimeCtx, w, &input); err != nil {
			if isBenignDebugSSECloseError(err) {
				log.Printf("DebugAgentSSE client disconnected client_ip=%s err=%v", c.ClientIP(), err)
			} else {
				log.Printf("DebugAgentSSE run failed client_ip=%s err=%v", c.ClientIP(), err)
			}
		}
		return false
	})
}

func isBenignDebugSSECloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET)
}
