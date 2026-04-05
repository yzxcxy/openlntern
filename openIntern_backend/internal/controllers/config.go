package controllers

import (
	"net/http"

	"openIntern/internal/config"
	"openIntern/internal/response"

	"github.com/gin-gonic/gin"
)

// GetConfig 获取当前配置
func GetConfig(c *gin.Context) {
	runtimeCfg := config.GetRuntimeConfig()
	response.JSONSuccess(c, http.StatusOK, runtimeCfg.ToResponse())
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Agent              map[string]interface{} `json:"agent,omitempty"`
	Tools              map[string]interface{} `json:"tools,omitempty"`
	ContextCompression map[string]interface{} `json:"context_compression,omitempty"`
	Plugin             map[string]interface{} `json:"plugin,omitempty"`
	SummaryLLM         map[string]interface{} `json:"summary_llm,omitempty"`
	COS                map[string]interface{} `json:"cos,omitempty"`
	APMPlus            map[string]interface{} `json:"apmplus,omitempty"`
}

// UpdateConfig 更新配置
func UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "invalid request: "+err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Agent != nil {
		updates["agent"] = req.Agent
	}
	if req.Tools != nil {
		updates["tools"] = req.Tools
	}
	if req.ContextCompression != nil {
		updates["context_compression"] = req.ContextCompression
	}
	if req.Plugin != nil {
		updates["plugin"] = req.Plugin
	}
	if req.SummaryLLM != nil {
		updates["summary_llm"] = req.SummaryLLM
	}
	if req.COS != nil {
		updates["cos"] = req.COS
	}
	if req.APMPlus != nil {
		updates["apmplus"] = req.APMPlus
	}

	if len(updates) == 0 {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "no config updates provided")
		return
	}

	if err := config.UpdateRuntimeConfig(updates); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to update config: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "config updated successfully")
}

// ReloadConfig 重新加载配置
func ReloadConfig(c *gin.Context) {
	if err := config.ReloadConfig(); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to reload config: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "config reloaded successfully")
}
