package controllers

import (
	"errors"
	"net/http"
	"openIntern/internal/response"
	modelsvc "openIntern/internal/services/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateModel(c *gin.Context) {
	var input modelsvc.CreateModelCatalogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := modelsvc.ModelCatalog.Create(c.GetString("user_id"), input)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	view, err := modelsvc.ModelCatalog.GetView(c.GetString("user_id"), item.ModelID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusCreated, view)
}

func ListModels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	providerID := c.Query("provider_id")
	items, total, err := modelsvc.ModelCatalog.List(c.GetString("user_id"), page, pageSize, keyword, providerID)
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

func GetModel(c *gin.Context) {
	view, err := modelsvc.ModelCatalog.GetView(c.GetString("user_id"), c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "model not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, view)
}

func UpdateModel(c *gin.Context) {
	var input modelsvc.UpdateModelCatalogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	err := modelsvc.ModelCatalog.Update(c.GetString("user_id"), c.Param("id"), input)
	if err != nil {
		status := http.StatusBadRequest
		code := response.CodeBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
			code = response.CodeNotFound
		}
		response.JSONError(c, status, code, err.Error())
		return
	}
	response.JSONMessage(c, http.StatusOK, "model updated successfully")
}

func DeleteModel(c *gin.Context) {
	err := modelsvc.ModelCatalog.Delete(c.GetString("user_id"), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		code := response.CodeBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
			code = response.CodeNotFound
		}
		response.JSONError(c, status, code, err.Error())
		return
	}
	response.JSONMessage(c, http.StatusOK, "model deleted successfully")
}

func ListModelCatalog(c *gin.Context) {
	items, err := modelsvc.ModelCatalog.ListCatalogOptions(c.GetString("user_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func GetDefaultModel(c *gin.Context) {
	item, err := modelsvc.DefaultModel.Get(c.GetString("user_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	if item == nil {
		response.JSONSuccess(c, http.StatusOK, gin.H{
			"config_key": modelsvc.SystemDefaultChatModelConfigKey,
			"model_id":   "",
		})
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}

func UpdateDefaultModel(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := modelsvc.DefaultModel.Set(c.GetString("user_id"), req.ModelID)
	if err != nil {
		status := http.StatusBadRequest
		code := response.CodeBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
			code = response.CodeNotFound
		}
		response.JSONError(c, status, code, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, item)
}
