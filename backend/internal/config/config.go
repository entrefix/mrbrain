package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabasePath   string
	JWTSecret      string
	JWTExpiration  time.Duration
	EncryptionKey  string
	OpenAIBaseURL  string
	OpenAIAPIKey   string
	OpenAIModel    string
	AllowedOrigins []string
	SearXNGURLs    []string
	// RAG/Embedding settings
	EmbeddingModel string
	VectorDBPath   string
	RAGEnabled     bool
	// NIM Embedding settings
	NIMAPIKey       string
	NIMBaseURL      string
	NIMModel        string
	NIMRPMLimit     int
	NIMEmbeddingDim int
}

func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	// Try current directory first, then parent directory
	godotenv.Load()
	godotenv.Load("../.env")

	jwtExp := os.Getenv("JWT_EXPIRATION")
	if jwtExp == "" {
		jwtExp = "24h"
	}

	expDuration, err := time.ParseDuration(jwtExp)
	if err != nil {
		expDuration = 24 * time.Hour
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8099"
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/todomyday.db"
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	origins := []string{"http://localhost:3111"}
	if allowedOrigins != "" {
		// Support comma-separated origins
		origins = strings.Split(allowedOrigins, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
	}

	openaiModel := os.Getenv("OPENAI_MODEL")
	if openaiModel == "" {
		openaiModel = "gpt-3.5-turbo"
	}

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		// Generate a default key for development (should be set in production)
		encryptionKey = "todomyday-dev-encryption-key-32"
	}

	// Parse SearXNG URLs (comma-separated for round-robin)
	var searxngURLs []string
	searxngEnv := os.Getenv("SEARXNG_URLS")
	if searxngEnv != "" {
		urls := strings.Split(searxngEnv, ",")
		for _, u := range urls {
			u = strings.TrimSpace(u)
			if u != "" {
				searxngURLs = append(searxngURLs, strings.TrimSuffix(u, "/"))
			}
		}
	}

	// RAG/Embedding settings
	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}

	vectorDBPath := os.Getenv("VECTOR_DB_PATH")
	if vectorDBPath == "" {
		vectorDBPath = "./data/vectors"
	}

	ragEnabled := os.Getenv("RAG_ENABLED") != "false" // Enabled by default if NIM is configured

	// NIM Embedding settings
	nimBaseURL := os.Getenv("NIM_BASE_URL")
	if nimBaseURL == "" {
		nimBaseURL = "https://integrate.api.nvidia.com/v1"
	}

	nimModel := os.Getenv("NIM_MODEL")
	if nimModel == "" {
		nimModel = "nvidia/nv-embedqa-e5-v5"
	}

	nimRPMLimit := 40
	if rpmStr := os.Getenv("NIM_RPM_LIMIT"); rpmStr != "" {
		if rpm, err := strconv.Atoi(rpmStr); err == nil && rpm > 0 {
			nimRPMLimit = rpm
		}
	}

	nimEmbeddingDim := 1024
	if dimStr := os.Getenv("NIM_EMBEDDING_DIM"); dimStr != "" {
		if dim, err := strconv.Atoi(dimStr); err == nil && dim > 0 {
			nimEmbeddingDim = dim
		}
	}

	return &Config{
		Port:            port,
		DatabasePath:    dbPath,
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTExpiration:   expDuration,
		EncryptionKey:   encryptionKey,
		OpenAIBaseURL:   os.Getenv("OPENAI_BASE_URL"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:     openaiModel,
		AllowedOrigins:  origins,
		SearXNGURLs:     searxngURLs,
		EmbeddingModel:  embeddingModel,
		VectorDBPath:    vectorDBPath,
		RAGEnabled:      ragEnabled,
		NIMAPIKey:       os.Getenv("NIM_API_KEY"),
		NIMBaseURL:      nimBaseURL,
		NIMModel:        nimModel,
		NIMRPMLimit:     nimRPMLimit,
		NIMEmbeddingDim: nimEmbeddingDim,
	}, nil
}
