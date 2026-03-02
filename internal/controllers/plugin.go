package controllers

import (
	"net/http"
	"openIntern/internal/response"
	"openIntern/internal/services"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ListPlugins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	items, total, err := services.Plugin.List(page, pageSize, services.PluginListFilter{
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
	item, err := services.Plugin.GetByPluginID(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func CreatePlugin(c *gin.Context) {
	var input services.UpsertPluginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := services.Plugin.Create(input)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusCreated, item)
}

func GetPluginDefaults(c *gin.Context) {
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"default_icon_url": services.GetDefaultPluginIconURL(),
	})
}

func UploadPluginIcon(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "only image files are supported")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	defer file.Close()

	ext := filepath.Ext(fileHeader.Filename)
	key := path.Join("plugin", "icon", uuid.NewString()+ext)
	url, err := services.File.UploadWithKey(c.Request.Context(), key, file, fileHeader)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"key": key,
		"url": url,
	})
}

func UpdatePlugin(c *gin.Context) {
	var input services.UpsertPluginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := services.Plugin.Update(strings.TrimSpace(c.Param("id")), input)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DebugCodePlugin(c *gin.Context) {
	var input services.CodeDebugInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}

	item, err := services.Plugin.DebugCodeTool(c.Request.Context(), input)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func EnablePlugin(c *gin.Context) {
	item, err := services.Plugin.SetEnabled(strings.TrimSpace(c.Param("id")), true)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DisablePlugin(c *gin.Context) {
	item, err := services.Plugin.SetEnabled(strings.TrimSpace(c.Param("id")), false)
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func SyncPlugin(c *gin.Context) {
	item, err := services.Plugin.Sync(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func DeletePlugin(c *gin.Context) {
	if err := services.Plugin.Delete(strings.TrimSpace(c.Param("id"))); err != nil {
		writePluginError(c, err)
		return
	}
	response.JSONMessage(c, http.StatusOK, "plugin deleted successfully")
}

func ListAvailablePluginsForChat(c *gin.Context) {
	items, err := services.Plugin.ListAvailableForChat()
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
