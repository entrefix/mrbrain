package main

import (
	"context"
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

	// Initialize RAG components (before todo/memory services so they can use it)
	var ragService *services.RAGService

	if cfg.RAGEnabled && cfg.OpenAIBaseURL != "" && cfg.OpenAIAPIKey != "" {
		log.Println("Initializing RAG service...")

		// Create embedding service
		embeddingService := services.NewEmbeddingService(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.EmbeddingModel)

		// Create FTS repository and initialize tables
		ftsRepo := repository.NewFTSRepository(db)
		if err := ftsRepo.InitFTSTables(); err != nil {
			log.Printf("Warning: Failed to initialize FTS tables: %v", err)
		} else {
			// Populate FTS from existing data
			if err := ftsRepo.PopulateFTSFromExisting(); err != nil {
				log.Printf("Warning: Failed to populate FTS: %v", err)
			}
		}

		// Create vector repository
		vectorRepo, err := repository.NewVectorRepository(
			repository.VectorConfig{
				PersistPath: cfg.VectorDBPath,
				Dimension:   embeddingService.GetDimension(),
			},
			func(ctx context.Context, text string) ([]float32, error) {
				return embeddingService.Embed(ctx, text)
			},
		)
		if err != nil {
			log.Printf("Warning: Failed to create vector repository: %v", err)
		} else {
			// Create RAG service
			ragService = services.NewRAGService(
				vectorRepo,
				ftsRepo,
				todoRepo,
				memoryRepo,
				embeddingService,
				aiService,
				aiProviderService,
			)
			log.Printf("RAG service initialized with embedding model: %s", cfg.EmbeddingModel)
		}
	} else {
		log.Println("RAG service not enabled - set OPENAI_BASE_URL and OPENAI_API_KEY to enable")
	}

	// Initialize todo and memory services (with RAG integration)
	todoService := services.NewTodoService(todoRepo, aiService, aiProviderService, ragService)
	memoryService := services.NewMemoryService(memoryRepo, todoRepo, aiService, aiProviderService, scraperService, ragService)

	// Setup router
	r := router.Setup(authService, todoService, groupService, aiProviderService, memoryService, ragService, cfg.AllowedOrigins)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("Allowed origins: %v", cfg.AllowedOrigins)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
