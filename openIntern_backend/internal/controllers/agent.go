package controllers

import (
	"errors"
	"net/http"
	"openIntern/internal/response"
	agentsvc "openIntern/internal/services/agent"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateAgent(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	var input agentsvc.UpsertAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := agentsvc.AgentDefinition.Create(c.Request.Context(), ownerID, input)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusCreated, item)
}

func ListAgents(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	items, total, err := agentsvc.AgentDefinition.List(ownerID, page, pageSize, agentsvc.AgentListFilter{
		Keyword:   c.Query("keyword"),
		AgentType: c.Query("agent_type"),
		Status:    c.Query("status"),
	})
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  items,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func GetAgent(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	item, err := agentsvc.AgentDefinition.Get(ownerID, strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func UpdateAgent(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	var input agentsvc.UpsertAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := agentsvc.AgentDefinition.Update(c.Request.Context(), ownerID, strings.TrimSpace(c.Param("id")), input)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func EnableAgent(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	item, err := agentsvc.AgentDefinition.SetEnabled(c.Request.Context(), ownerID, strings.TrimSpace(c.Param("id")), true)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DisableAgent(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	item, err := agentsvc.AgentDefinition.SetEnabled(c.Request.Context(), ownerID, strings.TrimSpace(c.Param("id")), false)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DeleteAgent(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	if err := agentsvc.AgentDefinition.Delete(ownerID, strings.TrimSpace(c.Param("id"))); err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONMessage(c, http.StatusOK, "agent deleted successfully")
}

func ListEnabledAgentOptions(c *gin.Context) {
	ownerID := strings.TrimSpace(c.GetString("user_id"))
	if ownerID == "" {
		response.Unauthorized(c)
		return
	}
	items, err := agentsvc.AgentDefinition.ListEnabledOptions(ownerID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func writeAgentError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.NotFound(c, "agent not found")
		return
	}
	message := strings.TrimSpace(err.Error())
	status := http.StatusBadRequest
	code := response.CodeBadRequest
	if strings.Contains(strings.ToLower(message), "not found") {
		status = http.StatusNotFound
		code = response.CodeNotFound
	}
	response.JSONError(c, status, code, message)
}
