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
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Access-Token, X-Token-Expires")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.POST("/v1/chat/sse", middleware.AuthRequired(), controllers.ChatSSE)

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

	threadGroup := r.Group("/v1/threads", middleware.AuthRequired())
	{
		threadGroup.GET("", controllers.ListThreads)
		threadGroup.GET("/:thread_id", controllers.GetThread)
		threadGroup.PUT("/:thread_id", controllers.UpdateThread)
		threadGroup.DELETE("/:thread_id", controllers.DeleteThread)
		threadGroup.GET("/:thread_id/messages", controllers.ListMessages)
	}

	// A2UI routes
	a2uiGroup := r.Group("/v1/a2uis", middleware.AuthRequired())
	{
		a2uiGroup.POST("", controllers.CreateA2UI)
		a2uiGroup.GET("", controllers.ListA2UIs)
		a2uiGroup.GET("/:id", controllers.GetA2UI)
		a2uiGroup.PUT("/:id", controllers.UpdateA2UI)
		a2uiGroup.DELETE("/:id", controllers.DeleteA2UI)
	}

	skillGroup := r.Group("/v1/skills", middleware.AuthRequired())
	{
		skillGroup.GET("", controllers.ListSkillFiles)
		skillGroup.POST("/import", controllers.ImportSkill)
		skillGroup.DELETE("/:name", controllers.DeleteSkill)
		skillGroup.GET("/content/:name", controllers.ReadSkillContent)
	}

	skillMetaGroup := r.Group("/v1/skills/meta", middleware.AuthRequired())
	{
		skillMetaGroup.POST("", controllers.CreateSkillMeta)
		skillMetaGroup.GET("", controllers.ListSkills)
		skillMetaGroup.GET("/:name", controllers.GetSkillMetaByName)
	}

	kbGroup := r.Group("/v1/kbs", middleware.AuthRequired())
	{
		kbGroup.GET("", controllers.ListKnowledgeBases)
		kbGroup.GET("/:name/tree", controllers.GetKnowledgeBaseTree)
		kbGroup.POST("/import", controllers.ImportKnowledgeBase)
		kbGroup.POST("/file", controllers.UploadKnowledgeBaseFile)
		kbGroup.POST("/move", controllers.MoveKnowledgeBaseEntry)
		kbGroup.POST("/drag", controllers.DragKnowledgeBaseEntry)
		kbGroup.DELETE("/:name", controllers.DeleteKnowledgeBase)
		kbGroup.DELETE("/entry", controllers.DeleteKnowledgeBaseEntry)
	}

	return r
}
