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

	// Validate required config for Supabase
	if cfg.SupabaseURL == "" {
		log.Fatal("SUPABASE_URL environment variable is required")
	}
	if cfg.SupabaseJWTSecret == "" {
		log.Fatal("SUPABASE_JWT_SECRET environment variable is required")
	}
	if cfg.SupabaseAnonKey == "" {
		log.Fatal("SUPABASE_ANON_KEY environment variable is required")
	}
	if cfg.SupabaseServiceRoleKey == "" {
		log.Fatal("SUPABASE_SERVICE_ROLE_KEY environment variable is required")
	}

	// Connect to database
	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("Connected to database: %s", cfg.DatabasePath)

	// === Database Health Checks ===

	// 1. Check WAL checkpoint status and recover if needed
	log.Println("Performing database health checks...")
	var walCheckpointResult string
	err = db.QueryRow("PRAGMA wal_checkpoint(PASSIVE)").Scan(&walCheckpointResult)
	if err != nil {
		log.Printf("Warning: WAL checkpoint failed: %v", err)
	} else {
		log.Printf("WAL checkpoint status: %s", walCheckpointResult)
	}

	// 2. Verify database integrity
	var integrityCheck string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&integrityCheck)
	if err != nil {
		log.Printf("Warning: Integrity check failed: %v", err)
	} else if integrityCheck != "ok" {
		log.Printf("⚠️  WARNING: Database integrity check failed: %s", integrityCheck)
	} else {
		log.Println("✅ Database integrity check passed")
	}

	// 3. Check critical table counts
	var userCount, todoCount, memoryCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&todoCount)
	db.QueryRow("SELECT COUNT(*) FROM memories").Scan(&memoryCount)
	log.Printf("Database stats - Users: %d, Todos: %d, Memories: %d", userCount, todoCount, memoryCount)

	// 4. Check for WAL file presence
	var journalMode string
	db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	log.Printf("Journal mode: %s", journalMode)

	log.Println("Database health checks complete")
	// === End Health Checks ===

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	todoRepo := repository.NewTodoRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	aiProviderRepo := repository.NewAIProviderRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)
	chatRepo := repository.NewChatRepository(db)

	// Initialize encryptor for API keys
	encryptor := crypto.NewEncryptor(cfg.EncryptionKey)

	// Initialize Supabase auth service
	supabaseAuthService := services.NewSupabaseAuthService(
		userRepo,
		cfg.SupabaseJWTSecret,
		cfg.SupabaseURL,
		cfg.SupabaseAnonKey,
		cfg.SupabaseServiceRoleKey,
	)

	// Initialize core services
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
	var vectorRepo *repository.VectorRepository

	if cfg.RAGEnabled && cfg.NIMAPIKey != "" {
		log.Println("Initializing RAG service with NVIDIA NIM embeddings...")

		// Create NIM embedding service
		embeddingService := services.NewEmbeddingService(
			cfg.NIMBaseURL,
			cfg.NIMAPIKey,
			cfg.NIMModel,
			cfg.NIMRPMLimit,
			cfg.NIMEmbeddingDim,
		)

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

		// Create vector repository (uses EmbedPassage for indexing, EmbedQuery for search)
		vRepo, err := repository.NewVectorRepository(
			repository.VectorConfig{
				PersistPath: cfg.VectorDBPath,
				Dimension:   embeddingService.GetDimension(),
			},
			embeddingService,
		)
		if err != nil {
			log.Printf("Warning: Failed to create vector repository: %v", err)
		} else {
			vectorRepo = vRepo
			// Create RAG service
			ragService = services.NewRAGService(
				vectorRepo,
				ftsRepo,
				todoRepo,
				memoryRepo,
				embeddingService,
				aiService,
				aiProviderService,
				scraperService,
			)
			log.Printf("RAG service initialized with NIM embedding model: %s (dim=%d, rpm=%d)",
				cfg.NIMModel, cfg.NIMEmbeddingDim, cfg.NIMRPMLimit)
		}
	} else {
		log.Println("RAG service not enabled - set NIM_API_KEY to enable")
	}

	// Initialize todo and memory services (with RAG integration)
	todoService := services.NewTodoService(todoRepo, aiService, aiProviderService, ragService)
	memoryService := services.NewMemoryService(memoryRepo, todoRepo, aiService, aiProviderService, scraperService, ragService)

	// Initialize user data service (for data management)
	userDataService := services.NewUserDataService(memoryRepo, todoRepo, groupRepo, vectorRepo, ragService)

	// Initialize file parser service
	fileParserService := services.NewFileParserService()

	// Initialize upload job service
	uploadJobService := services.NewUploadJobService()

	// Initialize chat service
	chatService := services.NewChatService(chatRepo)

	// Setup router
	r := router.Setup(supabaseAuthService, userRepo, todoService, groupService, aiProviderService, memoryService, ragService, userDataService, fileParserService, uploadJobService, chatService, cfg.AllowedOrigins)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("Allowed origins: %v", cfg.AllowedOrigins)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
