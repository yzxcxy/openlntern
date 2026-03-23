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
	r.POST("/v1/chat/uploads", middleware.AuthRequired(), controllers.UploadChatAsset)

	agentGroup := r.Group("/v1/agents", middleware.AuthRequired())
	{
		agentGroup.POST("", controllers.CreateAgent)
		agentGroup.GET("/debug/sse", controllers.DebugAgentSSEInfo)
		agentGroup.POST("/debug/sse", controllers.DebugAgentSSE)
		agentGroup.GET("", controllers.ListAgents)
		agentGroup.GET("/enabled-options", controllers.ListEnabledAgentOptions)
		agentGroup.GET("/:id", controllers.GetAgent)
		agentGroup.PUT("/:id", controllers.UpdateAgent)
		agentGroup.POST("/:id/enable", controllers.EnableAgent)
		agentGroup.POST("/:id/disable", controllers.DisableAgent)
		agentGroup.DELETE("/:id", controllers.DeleteAgent)
	}

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

	pluginGroup := r.Group("/v1/plugins", middleware.AuthRequired())
	{
		pluginGroup.GET("", controllers.ListPlugins)
		pluginGroup.POST("", controllers.CreatePlugin)
		pluginGroup.POST("/code/debug", controllers.DebugCodePlugin)
		pluginGroup.GET("/defaults", controllers.GetPluginDefaults)
		pluginGroup.POST("/icon", controllers.UploadPluginIcon)
		pluginGroup.GET("/available-for-chat", controllers.ListAvailablePluginsForChat)
		pluginGroup.GET("/:id", controllers.GetPlugin)
		pluginGroup.PUT("/:id", controllers.UpdatePlugin)
		pluginGroup.POST("/:id/enable", controllers.EnablePlugin)
		pluginGroup.POST("/:id/disable", controllers.DisablePlugin)
		pluginGroup.POST("/:id/sync", controllers.SyncPlugin)
		pluginGroup.DELETE("/:id", controllers.DeletePlugin)
	}

	kbGroup := r.Group("/v1/kbs", middleware.AuthRequired())
	{
		kbGroup.GET("", controllers.ListKnowledgeBases)
		kbGroup.GET("/entry/content", controllers.GetKnowledgeBaseContent)
		kbGroup.GET("/:name/tree", controllers.GetKnowledgeBaseTree)
		kbGroup.POST("/import", controllers.ImportKnowledgeBase)
		kbGroup.POST("/file", controllers.UploadKnowledgeBaseFile)
		kbGroup.POST("/move", controllers.MoveKnowledgeBaseEntry)
		kbGroup.POST("/drag", controllers.DragKnowledgeBaseEntry)
		kbGroup.DELETE("/entry", controllers.DeleteKnowledgeBaseEntry)
		kbGroup.DELETE("/:name", controllers.DeleteKnowledgeBase)
	}

	modelProviderGroup := r.Group("/v1/model-providers", middleware.AuthRequired())
	{
		modelProviderGroup.POST("", controllers.CreateModelProvider)
		modelProviderGroup.GET("", controllers.ListModelProviders)
		modelProviderGroup.GET("/:id", controllers.GetModelProvider)
		modelProviderGroup.PUT("/:id", controllers.UpdateModelProvider)
		modelProviderGroup.DELETE("/:id", controllers.DeleteModelProvider)
	}

	modelGroup := r.Group("/v1/models", middleware.AuthRequired())
	{
		modelGroup.POST("", controllers.CreateModel)
		modelGroup.GET("", controllers.ListModels)
		modelGroup.GET("/catalog", controllers.ListModelCatalog)
		modelGroup.GET("/default", controllers.GetDefaultModel)
		modelGroup.PUT("/default", controllers.UpdateDefaultModel)
		modelGroup.GET("/:id", controllers.GetModel)
		modelGroup.PUT("/:id", controllers.UpdateModel)
		modelGroup.DELETE("/:id", controllers.DeleteModel)
	}

	return r
}
