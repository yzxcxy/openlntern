package routers

import (
	"openIntern/internal/controllers"
	"openIntern/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.POST("/v1/chat/sse", controllers.ChatSSE)

	// User routes
	userGroup := r.Group("/v1/users", middleware.AuthRequired())
	{
		userGroup.POST("", controllers.CreateUser)
		userGroup.GET("", controllers.ListUsers)
		userGroup.GET("/:id", controllers.GetUser)
		userGroup.PUT("/:id", controllers.UpdateUser)
		userGroup.POST("/:id/avatar", controllers.UploadAvatar)
		userGroup.DELETE("/:id", controllers.DeleteUser)
	}

	authGroup := r.Group("/v1/auth")
	{
		authGroup.POST("/register", controllers.Register)
		authGroup.POST("/login", controllers.Login)
	}

	// A2UI routes
	a2uiGroup := r.Group("/v1/a2uis", middleware.AuthRequired())
	{
		a2uiGroup.POST("", controllers.CreateA2UI)
		a2uiGroup.GET("", controllers.ListA2UIs)
		a2uiGroup.GET("/official", controllers.ListOfficialA2UIs)
		a2uiGroup.GET("/custom", controllers.ListCustomA2UIs)
		a2uiGroup.GET("/:id", controllers.GetA2UI)
		a2uiGroup.PUT("/:id", controllers.UpdateA2UI)
		a2uiGroup.DELETE("/:id", controllers.DeleteA2UI)
	}

	return r
}
