package controllers

import (
	"errors"
	"net/http"
	"openIntern/internal/response"
	"openIntern/internal/services"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateModel(c *gin.Context) {
	var input services.CreateModelCatalogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := services.ModelCatalog.Create(input)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	view, err := services.ModelCatalog.GetView(item.ModelID)
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
	items, total, err := services.ModelCatalog.List(page, pageSize, keyword, providerID)
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
	view, err := services.ModelCatalog.GetView(c.Param("id"))
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
	var input services.UpdateModelCatalogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	err := services.ModelCatalog.Update(c.Param("id"), input)
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
	err := services.ModelCatalog.Delete(c.Param("id"))
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
	items, err := services.ModelCatalog.ListCatalogOptions()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func GetDefaultModel(c *gin.Context) {
	item, err := services.DefaultModel.Get()
	if err != nil {
		response.InternalError(c)
		return
	}
	if item == nil {
		response.JSONSuccess(c, http.StatusOK, gin.H{
			"config_key": services.SystemDefaultChatModelConfigKey,
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
	item, err := services.DefaultModel.Set(req.ModelID)
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
