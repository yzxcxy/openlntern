package routers

import (
	"agent_backend/internal/controllers"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", controllers.Ping)
	r.GET("/hello", controllers.Hello)
	r.POST("/v1/chat/sse", controllers.ChatSSE)

	return r
}
