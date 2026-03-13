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

func CreateModelProvider(c *gin.Context) {
	var input modelsvc.CreateModelProviderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	item, err := modelsvc.ModelProvider.Create(input)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusCreated, modelsvc.ModelProvider.ToView(item))
}

func ListModelProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	items, total, err := modelsvc.ModelProvider.List(page, pageSize, keyword)
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

func GetModelProvider(c *gin.Context) {
	item, err := modelsvc.ModelProvider.GetByProviderID(c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "model provider not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, modelsvc.ModelProvider.ToView(item))
}

func UpdateModelProvider(c *gin.Context) {
	var input modelsvc.UpdateModelProviderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c)
		return
	}
	err := modelsvc.ModelProvider.Update(c.Param("id"), input)
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
	response.JSONMessage(c, http.StatusOK, "model provider updated successfully")
}

func DeleteModelProvider(c *gin.Context) {
	err := modelsvc.ModelProvider.Delete(c.Param("id"))
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
	response.JSONMessage(c, http.StatusOK, "model provider deleted successfully")
}
