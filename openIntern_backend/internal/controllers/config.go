package controllers

import (
	"net/http"

	"openIntern/internal/config"
	"openIntern/internal/response"
	openvikingsvc "openIntern/internal/services/openviking"

	"github.com/gin-gonic/gin"
)

// GetConfig 获取当前配置
func GetConfig(c *gin.Context) {
	runtimeCfg := config.GetRuntimeConfig()
	ovCfg := config.GetOpenVikingServiceConfig()

	resp := runtimeCfg.ToResponse()
	resp.OpenVikingService = ovCfg.ToResponse()

	response.JSONSuccess(c, http.StatusOK, resp)
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Agent              map[string]interface{} `json:"agent,omitempty"`
	Tools              map[string]interface{} `json:"tools,omitempty"`
	ContextCompression map[string]interface{} `json:"context_compression,omitempty"`
	Plugin             map[string]interface{} `json:"plugin,omitempty"`
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

// UpdateOpenVikingServiceConfigRequest 更新 OpenViking 服务配置请求
type UpdateOpenVikingServiceConfigRequest struct {
	Storage   map[string]interface{} `json:"storage,omitempty"`
	Log       map[string]interface{} `json:"log,omitempty"`
	Embedding map[string]interface{} `json:"embedding,omitempty"`
	VLM       map[string]interface{} `json:"vlm,omitempty"`
	Parsers   map[string]interface{} `json:"parsers,omitempty"`
	Feishu    map[string]interface{} `json:"feishu,omitempty"`
	Rerank    map[string]interface{} `json:"rerank,omitempty"`
}

// UpdateOpenVikingServiceConfig 更新 OpenViking 服务配置
func UpdateOpenVikingServiceConfig(c *gin.Context) {
	var req UpdateOpenVikingServiceConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "invalid request: "+err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Storage != nil {
		updates["storage"] = req.Storage
	}
	if req.Log != nil {
		updates["log"] = req.Log
	}
	if req.Embedding != nil {
		updates["embedding"] = req.Embedding
	}
	if req.VLM != nil {
		updates["vlm"] = req.VLM
	}
	if req.Parsers != nil {
		updates["parsers"] = req.Parsers
	}
	if req.Feishu != nil {
		updates["feishu"] = req.Feishu
	}
	if req.Rerank != nil {
		updates["rerank"] = req.Rerank
	}

	if len(updates) == 0 {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "no config updates provided")
		return
	}

	if err := config.UpdateOpenVikingServiceConfig(updates); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to update openviking config: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "openviking config updated successfully")
}

// ReloadConfig 重新加载配置
func ReloadConfig(c *gin.Context) {
	if err := config.ReloadConfig(); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to reload config: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "config reloaded successfully")
}

// GetOpenVikingStatus 获取 OpenViking 服务状态
func GetOpenVikingStatus(c *gin.Context) {
	manager := openvikingsvc.GetManager()
	if manager == nil {
		response.JSONError(c, http.StatusServiceUnavailable, response.CodeInternal, "openviking manager not initialized")
		return
	}

	status := manager.Status()
	response.JSONSuccess(c, http.StatusOK, status)
}

// StartOpenViking 启动 OpenViking 服务
func StartOpenViking(c *gin.Context) {
	manager := openvikingsvc.GetManager()
	if manager == nil {
		response.JSONError(c, http.StatusServiceUnavailable, response.CodeInternal, "openviking manager not initialized")
		return
	}

	if err := manager.Start(); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to start openviking: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "openviking started successfully")
}

// StopOpenViking 停止 OpenViking 服务
func StopOpenViking(c *gin.Context) {
	manager := openvikingsvc.GetManager()
	if manager == nil {
		response.JSONError(c, http.StatusServiceUnavailable, response.CodeInternal, "openviking manager not initialized")
		return
	}

	if err := manager.Stop(); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to stop openviking: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "openviking stopped successfully")
}

// RestartOpenViking 重启 OpenViking 服务
func RestartOpenViking(c *gin.Context) {
	manager := openvikingsvc.GetManager()
	if manager == nil {
		response.JSONError(c, http.StatusServiceUnavailable, response.CodeInternal, "openviking manager not initialized")
		return
	}

	if err := manager.Restart(); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "failed to restart openviking: "+err.Error())
		return
	}

	response.JSONMessage(c, http.StatusOK, "openviking restarted successfully")
}