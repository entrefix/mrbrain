package config

import (
	"os"
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
	// RAG settings
	RAGEnabled bool
	// ClaraVector settings
	ClaraVectorURL    string
	ClaraVectorAPIKey string
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

	// RAG settings
	ragEnabled := os.Getenv("RAG_ENABLED") == "true"

	// ClaraVector settings
	claraVectorURL := os.Getenv("CLARAVECTOR_URL")
	claraVectorAPIKey := os.Getenv("CLARAVECTOR_API_KEY")

	return &Config{
		Port:              port,
		DatabasePath:      dbPath,
		JWTSecret:         os.Getenv("JWT_SECRET"),
		JWTExpiration:     expDuration,
		EncryptionKey:     encryptionKey,
		OpenAIBaseURL:     os.Getenv("OPENAI_BASE_URL"),
		OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:       openaiModel,
		AllowedOrigins:    origins,
		SearXNGURLs:       searxngURLs,
		RAGEnabled:        ragEnabled,
		ClaraVectorURL:    claraVectorURL,
		ClaraVectorAPIKey: claraVectorAPIKey,
	}, nil
}
