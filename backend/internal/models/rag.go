package models

import "time"

// EmbeddingType represents the type of embedding
type EmbeddingType string

const (
	// EmbeddingTypeDocument is used for stored content (todos, memories)
	EmbeddingTypeDocument EmbeddingType = "document"
	// EmbeddingTypeQuery is used for search queries
	EmbeddingTypeQuery EmbeddingType = "query"
)

// ContentType represents what type of content is being indexed
type ContentType string

const (
	ContentTypeTodo   ContentType = "todo"
	ContentTypeMemory ContentType = "memory"
)

// EmbeddingDimensions for different models
const (
	// OpenAI text-embedding-3-small
	DimensionOpenAISmall = 1536
	// OpenAI text-embedding-3-large
	DimensionOpenAILarge = 3072
	// Default dimension (OpenAI small)
	DimensionDefault = DimensionOpenAISmall
)

// Document represents a piece of content to be indexed/searched
type Document struct {
	ID          string            `json:"id"`
	ContentType ContentType       `json:"content_type"`
	ContentID   string            `json:"content_id"`
	UserID      string            `json:"user_id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata"`
	Embedding   []float32         `json:"embedding,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SearchRequest represents a search/Q&A request
type SearchRequest struct {
	Query        string   `json:"query" binding:"required"`
	ContentTypes []string `json:"content_types"` // Filter by type: todo, memory
	Limit        int      `json:"limit"`
	VectorWeight float64  `json:"vector_weight"` // 0-1, weight for vector vs keyword search
}

// SearchResult represents a single search result
type SearchResult struct {
	Document   *Document `json:"document"`
	Score      float64   `json:"score"`
	MatchType  string    `json:"match_type"` // "vector", "keyword", "hybrid"
	Highlights []string  `json:"highlights,omitempty"`
}

// SearchResponse contains search results
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	Query      string         `json:"query"`
	TotalCount int            `json:"total_count"`
	TimeTaken  float64        `json:"time_taken_ms"`
}

// AskRequest represents a Q&A request
type AskRequest struct {
	Question     string   `json:"question" binding:"required"`
	ContentTypes []string `json:"content_types"`
	MaxContext   int      `json:"max_context"` // Max docs to include in context
}

// AskResponse contains the answer and sources
type AskResponse struct {
	Answer    string         `json:"answer"`
	Sources   []SearchResult `json:"sources"`
	Question  string         `json:"question"`
	TimeTaken float64        `json:"time_taken_ms"`
}

// IndexStats provides statistics about the vector index
type IndexStats struct {
	TotalDocuments int            `json:"total_documents"`
	ByContentType  map[string]int `json:"by_content_type"`
	ByUser         map[string]int `json:"by_user"`
	LastIndexedAt  *time.Time     `json:"last_indexed_at"`
}

// IndexRequest for triggering indexing
type IndexRequest struct {
	ContentType ContentType `json:"content_type"`
	ContentID   string      `json:"content_id"`
	Reindex     bool        `json:"reindex"` // Force reindex even if already indexed
}

// IndexResponse after indexing
type IndexResponse struct {
	Indexed   int    `json:"indexed"`
	Skipped   int    `json:"skipped"`
	Errors    int    `json:"errors"`
	TimeTaken float64 `json:"time_taken_ms"`
}

// EmbeddingRequest for generating embeddings
type EmbeddingRequest struct {
	Texts []string      `json:"texts"`
	Type  EmbeddingType `json:"type"`
}

// EmbeddingResponse contains generated embeddings
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimension  int         `json:"dimension"`
}
