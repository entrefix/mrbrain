package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/handlers"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/repository"
	"github.com/todomyday/backend/internal/services"
)

func Setup(
	supabaseAuthService *services.SupabaseAuthService,
	userRepo *repository.UserRepository,
	todoService *services.TodoService,
	groupService *services.GroupService,
	aiProviderService *services.AIProviderService,
	memoryService *services.MemoryService,
	ragService *services.RAGService,
	userDataService *services.UserDataService,
	fileParserService *services.FileParserService,
	uploadJobService *services.UploadJobService,
	visionService *services.VisionService,
	chatService *services.ChatService,
	allowedOrigins []string,
) *gin.Engine {
	r := gin.Default()

	// Configure CORS
	corsConfig := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	r.Use(cors.New(corsConfig))

	// Health check
	r.GET("/health", handlers.HealthCheck)

	// Create handlers
	authHandler := handlers.NewAuthHandler(userRepo)
	todoHandler := handlers.NewTodoHandler(todoService)
	groupHandler := handlers.NewGroupHandler(groupService)
	aiProviderHandler := handlers.NewAIProviderHandler(aiProviderService)
	memoryHandler := handlers.NewMemoryHandler(memoryService, fileParserService, uploadJobService, visionService)
	ragHandler := handlers.NewRAGHandler(ragService)
	userDataHandler := handlers.NewUserDataHandler(userDataService)
	chatHandler := handlers.NewChatHandler(chatService)

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			// Register and Login are now handled by Supabase on the frontend
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(supabaseAuthService))
		{
			// Auth - get current user
			protected.GET("/auth/me", authHandler.Me)

			// Todos
			protected.GET("/todos", todoHandler.GetAll)
			protected.POST("/todos", todoHandler.Create)
			protected.GET("/todos/:id", todoHandler.GetByID)
			protected.PUT("/todos/:id", todoHandler.Update)
			protected.DELETE("/todos/:id", todoHandler.Delete)
			protected.PUT("/todos/reorder", todoHandler.Reorder)

			// Groups
			protected.GET("/groups", groupHandler.GetAll)
			protected.POST("/groups", groupHandler.Create)
			protected.GET("/groups/:id", groupHandler.GetByID)
			protected.PUT("/groups/:id", groupHandler.Update)
			protected.DELETE("/groups/:id", groupHandler.Delete)

			// AI Providers
			protected.GET("/ai-providers", aiProviderHandler.GetAll)
			protected.POST("/ai-providers", aiProviderHandler.Create)
			protected.GET("/ai-providers/:id", aiProviderHandler.GetByID)
			protected.PUT("/ai-providers/:id", aiProviderHandler.Update)
			protected.DELETE("/ai-providers/:id", aiProviderHandler.Delete)
			protected.POST("/ai-providers/test", aiProviderHandler.TestConnection)
			protected.POST("/ai-providers/:id/fetch-models", aiProviderHandler.FetchModels)
			protected.GET("/ai-providers/:id/models", aiProviderHandler.GetModels)

			// Memories
			protected.GET("/memories", memoryHandler.GetAll)
			protected.POST("/memories", memoryHandler.Create)
			protected.POST("/memories/upload", memoryHandler.UploadMemoryFile)
			protected.POST("/memories/upload-image", memoryHandler.UploadImage)
			protected.GET("/memories/upload/jobs/:job_id", memoryHandler.GetUploadJobStatus)
			protected.GET("/memories/categories", memoryHandler.GetCategories)
			protected.GET("/memories/category/:category", memoryHandler.GetByCategory)
			protected.GET("/memories/stats", memoryHandler.GetStats)
			protected.POST("/memories/search", memoryHandler.Search)
			protected.PUT("/memories/reorder", memoryHandler.Reorder)
			protected.GET("/memories/digest", memoryHandler.GetDigest)
			protected.POST("/memories/digest/generate", memoryHandler.GenerateDigest)
			protected.POST("/memories/web-search", memoryHandler.WebSearch)
			protected.GET("/memories/:id", memoryHandler.GetByID)
			protected.PUT("/memories/:id", memoryHandler.Update)
			protected.DELETE("/memories/:id", memoryHandler.Delete)
			protected.POST("/memories/:id/to-todo", memoryHandler.ConvertToTodo)

			// RAG - Search & Q&A
			protected.POST("/rag/search", ragHandler.Search)
			protected.POST("/rag/ask", ragHandler.Ask)
			protected.POST("/rag/index", ragHandler.IndexAll)
			protected.GET("/rag/stats", ragHandler.GetStats)

			// User Data Management
			protected.GET("/user/data/stats", userDataHandler.GetDataStats)
			protected.POST("/user/data/clear-memories", userDataHandler.ClearMemories)
			protected.POST("/user/data/clear-all", userDataHandler.ClearAllData)

			// Chat Threads
			protected.GET("/chat/threads/active", chatHandler.GetActiveThread)
			protected.GET("/chat/threads", chatHandler.GetAllThreads)
			protected.GET("/chat/threads/:id", chatHandler.GetThread)
			protected.POST("/chat/threads", chatHandler.CreateThread)
			protected.POST("/chat/threads/:id/messages", chatHandler.AddMessage)
			protected.DELETE("/chat/threads/:id", chatHandler.DeleteThread)
		}
	}

	return r
}
