package controllers

import (
	"net/http"
	"openIntern/internal/models"
	"openIntern/internal/response"
	accountsvc "openIntern/internal/services/account"
	storagesvc "openIntern/internal/services/storage"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type userResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// buildUserResponse keeps the external user payload stable while the stored avatar value becomes an object key.
func buildUserResponse(user *models.User) (*userResponse, error) {
	if user == nil {
		return nil, nil
	}
	avatarURL, err := storagesvc.ObjectStorage.ResolveObjectAccessURL(user.Avatar)
	if err != nil {
		return nil, err
	}
	return &userResponse{
		UserID:    user.UserID,
		Username:  user.Username,
		Email:     user.Email,
		Phone:     user.Phone,
		Avatar:    avatarURL,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

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
	userPayload, err := buildUserResponse(user)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"user":       userPayload,
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
	userPayload, err := buildUserResponse(user)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, userPayload)
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

	uploaded, err := storagesvc.ObjectStorage.UploadUserObject(c.Request.Context(), userID, storagesvc.UploadUserObjectSpec{
		Purpose:          storagesvc.ObjectPurposeAvatar,
		OriginalFileName: strings.TrimSpace(fileHeader.Filename),
		ContentType:      detectedContentType,
	}, file, fileHeader.Size)
	if err != nil {
		response.InternalError(c)
		return
	}
	if err := accountsvc.User.UpdateUser(userID, map[string]interface{}{"avatar": uploaded.Key}); err != nil {
		response.InternalError(c)
		return
	}
	avatarURL, err := storagesvc.ObjectStorage.ResolveObjectAccessURL(uploaded.Key)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"key": uploaded.Key,
		"url": avatarURL,
	})
}
