package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/todomyday/backend/internal/models"
)

// InputType represents the type of input for NIM embeddings
type InputType string

const (
	// InputTypePassage is used for documents/passages being indexed
	InputTypePassage InputType = "passage"
	// InputTypeQuery is used for search queries
	InputTypeQuery InputType = "query"
)

// EmbeddingService handles generating embeddings using NVIDIA NIM API
type EmbeddingService struct {
	baseURL     string
	apiKey      string
	model       string
	dimension   int
	minInterval time.Duration
	client      *http.Client

	// Rate limiting
	mu              sync.Mutex
	lastRequestTime time.Time
}

// NIM embedding request type
type nimEmbeddingRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	InputType      string `json:"input_type"`
	EncodingFormat string `json:"encoding_format"`
}

// NIM embedding response type
type nimEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// NewEmbeddingService creates a new NIM embedding service
func NewEmbeddingService(baseURL, apiKey, model string, rpmLimit, dimension int) *EmbeddingService {
	if model == "" {
		model = "nvidia/nv-embedqa-e5-v5"
	}

	if dimension <= 0 {
		dimension = models.DimensionNIM
	}

	if rpmLimit <= 0 {
		rpmLimit = 40
	}

	// Calculate minimum interval between requests (60 seconds / RPM limit)
	minInterval := time.Duration(float64(time.Minute) / float64(rpmLimit))

	return &EmbeddingService{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		apiKey:      apiKey,
		model:       model,
		dimension:   dimension,
		minInterval: minInterval,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if the service is properly configured
func (s *EmbeddingService) IsConfigured() bool {
	return s.baseURL != "" && s.apiKey != ""
}

// GetDimension returns the embedding dimension for the configured model
func (s *EmbeddingService) GetDimension() int {
	return s.dimension
}

// GetModel returns the configured model name
func (s *EmbeddingService) GetModel() string {
	return s.model
}

// rateLimit enforces rate limiting
func (s *EmbeddingService) rateLimit() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(s.lastRequestTime)

	if elapsed < s.minInterval {
		sleepDuration := s.minInterval - elapsed
		time.Sleep(sleepDuration)
	}

	s.lastRequestTime = time.Now()
}

// Embed generates an embedding for a single text (defaults to passage type)
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	return s.EmbedWithType(ctx, text, InputTypePassage)
}

// EmbedQuery generates an embedding optimized for search queries
func (s *EmbeddingService) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return s.EmbedWithType(ctx, text, InputTypeQuery)
}

// EmbedPassage generates an embedding optimized for document passages
func (s *EmbeddingService) EmbedPassage(ctx context.Context, text string) ([]float32, error) {
	return s.EmbedWithType(ctx, text, InputTypePassage)
}

// EmbedWithType generates an embedding with the specified input type
func (s *EmbeddingService) EmbedWithType(ctx context.Context, text string, inputType InputType) ([]float32, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("embedding service not configured")
	}

	// Sanitize the text
	text = SanitizeText(text)

	if text == "" || len(text) < 10 {
		return nil, fmt.Errorf("text too short or empty after sanitization")
	}

	// Truncate if too long
	text = TruncateForEmbedding(text)

	// Enforce rate limiting
	s.rateLimit()

	log.Printf("[Embedding] Generating embedding using model %s (type: %s, len: %d)",
		s.model, inputType, len(text))

	reqBody := nimEmbeddingRequest{
		Model:          s.model,
		Input:          text,
		InputType:      string(inputType),
		EncodingFormat: "float",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := s.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Embedding] Error response: %s", string(body))
		log.Printf("[Embedding] Failed text (first 200 chars): %s", truncateString(text, 200))
		return nil, fmt.Errorf("NIM API error: %s - %s", resp.Status, string(body))
	}

	var embeddingResp nimEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	embedding := embeddingResp.Data[0].Embedding
	log.Printf("[Embedding] Successfully generated embedding (dimension: %d, tokens: %d)",
		len(embedding), embeddingResp.Usage.TotalTokens)

	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts (one at a time with rate limiting)
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string, inputType InputType) ([][]float32, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("embedding service not configured")
	}

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := s.EmbedWithType(ctx, text, inputType)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// HealthCheck checks if NIM API is accessible
func (s *EmbeddingService) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := s.baseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// PrepareDocumentText creates a text representation of a document for embedding
func PrepareDocumentText(doc *models.Document) string {
	var parts []string

	if doc.Title != "" {
		parts = append(parts, doc.Title)
	}

	if doc.Content != "" {
		parts = append(parts, doc.Content)
	}

	// Add relevant metadata
	if doc.Metadata != nil {
		if category, ok := doc.Metadata["category"]; ok && category != "" {
			parts = append(parts, "Category: "+category)
		}
		if tags, ok := doc.Metadata["tags"]; ok && tags != "" {
			parts = append(parts, "Tags: "+tags)
		}
	}

	return strings.Join(parts, "\n")
}

// ChunkText splits long text into chunks for embedding using token-aware chunking
func ChunkText(text string, maxChunkSize int) []string {
	chunker := NewDocumentChunker(&ChunkerConfig{
		MaxTokens: 450,
	})

	chunks := chunker.ChunkText(text)
	result := make([]string, len(chunks))
	for i, chunk := range chunks {
		result[i] = chunk.Text
	}

	return result
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
