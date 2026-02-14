package routers

import (
	"openIntern/internal/controllers"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/v1/chat/sse", controllers.ChatSSE)

	return r
}
