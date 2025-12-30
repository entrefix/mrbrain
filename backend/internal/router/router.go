package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/handlers"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/services"
)

func Setup(
	authService *services.AuthService,
	todoService *services.TodoService,
	groupService *services.GroupService,
	aiProviderService *services.AIProviderService,
	memoryService *services.MemoryService,
	ragService *services.RAGService,
	userDataService *services.UserDataService,
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
	authHandler := handlers.NewAuthHandler(authService)
	todoHandler := handlers.NewTodoHandler(todoService)
	groupHandler := handlers.NewGroupHandler(groupService)
	aiProviderHandler := handlers.NewAIProviderHandler(aiProviderService)
	memoryHandler := handlers.NewMemoryHandler(memoryService)
	ragHandler := handlers.NewRAGHandler(ragService)
	userDataHandler := handlers.NewUserDataHandler(userDataService)

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
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
			protected.GET("/memories/categories", memoryHandler.GetCategories)
			protected.GET("/memories/category/:category", memoryHandler.GetByCategory)
			protected.GET("/memories/stats", memoryHandler.GetStats)
			protected.POST("/memories/search", memoryHandler.Search)
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
		}
	}

	return r
}
