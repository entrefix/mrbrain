package main

import (
	"log"

	"github.com/todomyday/backend/internal/config"
	"github.com/todomyday/backend/internal/crypto"
	"github.com/todomyday/backend/internal/database"
	"github.com/todomyday/backend/internal/repository"
	"github.com/todomyday/backend/internal/router"
	"github.com/todomyday/backend/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate required config
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Connect to database
	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("Connected to database: %s", cfg.DatabasePath)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	todoRepo := repository.NewTodoRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	aiProviderRepo := repository.NewAIProviderRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)

	// Initialize encryptor for API keys
	encryptor := crypto.NewEncryptor(cfg.EncryptionKey)

	// Initialize core services
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	aiService := services.NewAIService(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel)
	aiProviderService := services.NewAIProviderService(aiProviderRepo, encryptor)
	groupService := services.NewGroupService(groupRepo)

	// Initialize scraper service (optional - for web search)
	var scraperService *services.ScraperService
	if len(cfg.SearXNGURLs) > 0 {
		scraperService = services.NewScraperService(cfg.SearXNGURLs)
		log.Printf("Web search enabled via SearXNG: %v", cfg.SearXNGURLs)
	}

	// Log AI configuration status
	if aiService.IsConfigured() {
		log.Printf("AI service configured with model: %s", cfg.OpenAIModel)
	} else {
		log.Println("AI service not configured - todos will use original titles")
	}

	// Initialize RAG service with ClaraVector
	var ragService *services.RAGService

	if cfg.RAGEnabled && cfg.ClaraVectorURL != "" {
		log.Println("Initializing RAG service with ClaraVector...")

		// Create ClaraVector client
		claraClient := services.NewClaraVectorClient(cfg.ClaraVectorURL, cfg.ClaraVectorAPIKey)

		// Check health
		if err := claraClient.HealthCheck(); err != nil {
			log.Printf("Warning: ClaraVector health check failed: %v", err)
		} else {
			log.Println("ClaraVector service is healthy")
		}

		// Create RAG service
		ragService = services.NewRAGService(
			claraClient,
			todoRepo,
			memoryRepo,
			aiService,
			aiProviderService,
		)
		log.Printf("RAG service initialized with ClaraVector: %s", cfg.ClaraVectorURL)
	} else {
		log.Println("RAG service not enabled - set RAG_ENABLED=true and CLARAVECTOR_URL to enable")
	}

	// Initialize todo and memory services (with RAG integration)
	todoService := services.NewTodoService(todoRepo, aiService, aiProviderService, ragService)
	memoryService := services.NewMemoryService(memoryRepo, todoRepo, aiService, aiProviderService, scraperService, ragService)

	// Initialize user data service (for data management)
	userDataService := services.NewUserDataService(memoryRepo, todoRepo, groupRepo, ragService)

	// Initialize file parser service
	fileParserService := services.NewFileParserService()

	// Setup router
	r := router.Setup(authService, todoService, groupService, aiProviderService, memoryService, ragService, userDataService, fileParserService, cfg.AllowedOrigins)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("Allowed origins: %v", cfg.AllowedOrigins)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
