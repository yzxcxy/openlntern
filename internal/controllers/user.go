package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/response"
	"openIntern/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		response.BadRequest(c)
		return
	}

	if err := services.User.CreateUser(&user); err != nil {
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
	if err := services.User.CreateUser(&user); err != nil {
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
	user, err := services.User.Authenticate(req.Identifier, req.Password)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	token, expiresAt, err := services.GenerateToken(user.UserID, user.Role)
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
	user, err := services.User.GetUserByUserID(userID)
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

	if err := services.User.UpdateUser(userID, updates); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONMessage(c, http.StatusOK, "user updated successfully")
}

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if err := services.User.DeleteUser(userID); err != nil {
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "user deleted successfully")
}

// ListUsers 获取用户列表
func ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, total, err := services.User.ListUsers(page, pageSize)
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
