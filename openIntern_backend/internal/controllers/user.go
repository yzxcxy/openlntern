package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/response"
	accountsvc "openIntern/internal/services/account"
	storagesvc "openIntern/internal/services/storage"
	"path"
	"path/filepath"
	"strconv"

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
	token, expiresAt, err := accountsvc.GenerateToken(user.UserID, "")
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

// GetUser 获取用户
func GetUser(c *gin.Context) {
	userID := c.Param("id")
	user, err := accountsvc.User.GetUserByUserID(userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.JSONSuccess(c, http.StatusOK, user)
}

// UpdateUser 更新用户
func UpdateUser(c *gin.Context) {
	userID := c.Param("id")
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

func UploadAvatar(c *gin.Context) {
	userID := c.Param("id")
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

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if err := accountsvc.User.DeleteUser(userID); err != nil {
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "user deleted successfully")
}

// ListUsers 获取用户列表
func ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, total, err := accountsvc.User.ListUsers(page, pageSize)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  users,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}
