package controllers

import (
	"net/http"
	"openIntern/internal/response"
	pluginsvc "openIntern/internal/services/plugin"
	storagesvc "openIntern/internal/services/storage"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func ListPlugins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	items, total, err := pluginsvc.Plugin.List(c.GetString("user_id"), page, pageSize, pluginsvc.PluginListFilter{
		Source:      c.Query("source"),
		RuntimeType: c.Query("runtime_type"),
		Status:      c.Query("status"),
		Keyword:     c.Query("keyword"),
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  items,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func GetPlugin(c *gin.Context) {
	item, err := pluginsvc.Plugin.GetByPluginID(c.GetString("user_id"), strings.TrimSpace(c.Param("id")))
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func CreatePlugin(c *gin.Context) {
	var input pluginsvc.UpsertPluginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := pluginsvc.Plugin.Create(c.GetString("user_id"), input)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusCreated, item)
}

func GetPluginDefaults(c *gin.Context) {
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"default_icon_url": pluginsvc.GetDefaultPluginIconURL(),
	})
}

func UploadPluginIcon(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	detectedContentType, err := detectImageContentType(fileHeader)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "only image files are supported")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	defer file.Close()

	uploaded, err := storagesvc.ObjectStorage.UploadUserObject(c.Request.Context(), strings.TrimSpace(c.GetString("user_id")), storagesvc.UploadUserObjectSpec{
		Purpose:          storagesvc.ObjectPurposePlugin,
		ScopeSegments:    []string{"icon"},
		OriginalFileName: strings.TrimSpace(fileHeader.Filename),
		ContentType:      detectedContentType,
	}, file, fileHeader.Size)
	if err != nil {
		response.InternalError(c)
		return
	}
	accessURL, err := storagesvc.ObjectStorage.ResolveObjectAccessURL(uploaded.Key)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"key": uploaded.Key,
		"url": accessURL,
	})
}

func UpdatePlugin(c *gin.Context) {
	var input pluginsvc.UpsertPluginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := pluginsvc.Plugin.Update(c.GetString("user_id"), strings.TrimSpace(c.Param("id")), input)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DebugCodePlugin(c *gin.Context) {
	var input pluginsvc.CodeDebugInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}

	item, err := pluginsvc.Plugin.DebugCodeTool(c.Request.Context(), input)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func EnablePlugin(c *gin.Context) {
	item, err := pluginsvc.Plugin.SetEnabled(c.GetString("user_id"), strings.TrimSpace(c.Param("id")), true)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DisablePlugin(c *gin.Context) {
	item, err := pluginsvc.Plugin.SetEnabled(c.GetString("user_id"), strings.TrimSpace(c.Param("id")), false)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func SyncPlugin(c *gin.Context) {
	item, err := pluginsvc.Plugin.Sync(c.GetString("user_id"), strings.TrimSpace(c.Param("id")))
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DeletePlugin(c *gin.Context) {
	if err := pluginsvc.Plugin.Delete(c.GetString("user_id"), strings.TrimSpace(c.Param("id"))); err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONMessage(c, http.StatusOK, "plugin deleted successfully")
}

func ListAvailablePluginsForChat(c *gin.Context) {
	items, err := pluginsvc.Plugin.ListAvailableForChat(c.GetString("user_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func writePluginError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	status := http.StatusBadRequest
	code := response.CodeBadRequest
	switch {
	case strings.Contains(message, "not found"):
		status = http.StatusNotFound
		code = response.CodeNotFound
	case strings.Contains(message, "read-only"):
		status = http.StatusForbidden
		code = response.CodeForbidden
	case strings.Contains(message, "duplicate"):
		status = http.StatusBadRequest
		code = response.CodeBadRequest
	}
	response.JSONError(c, status, code, message)
}
