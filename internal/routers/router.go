package routers

import (
	"openIntern/internal/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/v1/chat/sse", controllers.ChatSSE)

	// User routes
	userGroup := r.Group("/v1/users")
	{
		userGroup.POST("", controllers.CreateUser)
		userGroup.GET("", controllers.ListUsers)
		userGroup.GET("/:id", controllers.GetUser)
		userGroup.PUT("/:id", controllers.UpdateUser)
		userGroup.DELETE("/:id", controllers.DeleteUser)
	}

	// A2UI routes
	a2uiGroup := r.Group("/v1/a2uis")
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
