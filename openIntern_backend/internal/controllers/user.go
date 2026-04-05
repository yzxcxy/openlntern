package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/response"
	accountsvc "openIntern/internal/services/account"
	storagesvc "openIntern/internal/services/storage"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		response.BadRequest(c)
		return
	}

	if err := accountsvc.User.CreateUser(&user); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusCreated, user)
}

func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c)
		return
	}
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	}
	if err := accountsvc.User.CreateUser(&user); err != nil {
		response.BadRequest(c)
		return
	}
	response.JSONSuccess(c, http.StatusCreated, user)
}

func Login(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c)
		return
	}
	user, err := accountsvc.User.Authenticate(req.Identifier, req.Password)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	token, expiresAt, err := accountsvc.GenerateToken(user.UserID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"user":       user,
	})
}

// GetCurrentUser 获取当前登录用户
func GetCurrentUser(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	user, err := accountsvc.User.GetUserByUserID(userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.JSONSuccess(c, http.StatusOK, user)
}

// UpdateCurrentUser 更新当前登录用户
func UpdateCurrentUser(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c)
		return
	}

	if err := accountsvc.User.UpdateUser(userID, updates); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONMessage(c, http.StatusOK, "user updated successfully")
}

func UploadCurrentUserAvatar(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	defer file.Close()

	ext := filepath.Ext(fileHeader.Filename)
	key := path.Join("avatar", userID, uuid.NewString()+ext)
	url, err := storagesvc.File.UploadWithKey(c.Request.Context(), key, file, fileHeader)
	if err != nil {
		response.InternalError(c)
		return
	}
	if err := accountsvc.User.UpdateUser(userID, map[string]interface{}{"avatar": url}); err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"key": key,
		"url": url,
	})
}
