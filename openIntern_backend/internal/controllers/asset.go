package controllers

import (
	"net/http"
	"openIntern/internal/response"
	storagesvc "openIntern/internal/services/storage"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetObjectAsset serves user and public objects through application URLs.
func GetObjectAsset(c *gin.Context) {
	objectKey := strings.TrimPrefix(strings.TrimSpace(c.Param("objectKey")), "/")
	if objectKey == "" {
		response.BadRequest(c)
		return
	}
	if strings.HasPrefix(objectKey, "users/") {
		expiresAt, err := strconv.ParseInt(strings.TrimSpace(c.Query("expires")), 10, 64)
		if err != nil || !storagesvc.ObjectStorage.VerifyObjectAccessSignature(objectKey, expiresAt, c.Query("signature")) {
			response.Forbidden(c)
			return
		}
	} else if !strings.HasPrefix(objectKey, "public/") {
		response.BadRequest(c)
		return
	}
	result, err := storagesvc.ObjectStorage.ReadObject(c.Request.Context(), objectKey)
	if err != nil {
		response.NotFound(c, "object not found")
		return
	}
	defer result.Reader.Close()
	c.DataFromReader(http.StatusOK, result.Size, result.ContentType, result.Reader, nil)
}
