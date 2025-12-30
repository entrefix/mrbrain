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
	"time"

	"github.com/todomyday/backend/internal/models"
)

// EmbeddingService handles generating embeddings from various AI providers
type EmbeddingService struct {
	baseURL   string
	apiKey    string
	model     string
	dimension int
	client    *http.Client
}

// OpenAI embedding request/response types
type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
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

// NewEmbeddingService creates a new embedding service with default config
func NewEmbeddingService(baseURL, apiKey, model string) *EmbeddingService {
	if model == "" {
		model = "text-embedding-3-small"
	}

	dimension := models.DimensionDefault
	if strings.Contains(model, "3-large") {
		dimension = models.DimensionOpenAILarge
	}

	return &EmbeddingService{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		apiKey:    apiKey,
		model:     model,
		dimension: dimension,
		client: &http.Client{
			Timeout: 60 * time.Second,
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

// Embed generates an embedding for a single text
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := s.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("embedding service not configured")
	}

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Clean and prepare texts
	cleanedTexts := make([]string, len(texts))
	for i, text := range texts {
		// Truncate long texts (OpenAI has a token limit)
		if len(text) > 8000 {
			text = text[:8000]
		}
		// Replace newlines with spaces for better embedding
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.TrimSpace(text)
		if text == "" {
			text = " " // OpenAI doesn't accept empty strings
		}
		cleanedTexts[i] = text
	}

	log.Printf("[Embedding] Generating embeddings for %d texts using model %s", len(texts), s.model)

	reqBody := openAIEmbeddingRequest{
		Model: s.model,
		Input: cleanedTexts,
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
		return nil, fmt.Errorf("embedding API error: %s - %s", resp.Status, string(body))
	}

	var embeddingResp openAIEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(embeddingResp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(embeddingResp.Data))
	}

	// Sort by index to ensure correct order
	embeddings := make([][]float32, len(texts))
	for _, data := range embeddingResp.Data {
		if data.Index >= 0 && data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	log.Printf("[Embedding] Successfully generated %d embeddings (dimension: %d, tokens: %d)",
		len(embeddings), len(embeddings[0]), embeddingResp.Usage.TotalTokens)

	return embeddings, nil
}

// EmbedWithProvider generates embeddings using a specific provider config
func (s *EmbeddingService) EmbedWithProvider(ctx context.Context, texts []string, config *AIProviderConfig) ([][]float32, error) {
	if config == nil || config.BaseURL == "" || config.APIKey == "" {
		return nil, fmt.Errorf("invalid provider config")
	}

	// For now, we only support OpenAI-compatible embedding APIs
	// Anthropic and Google have different embedding endpoints
	switch config.ProviderType {
	case models.ProviderTypeOpenAI, models.ProviderTypeCustom:
		// Create a temporary service with the provider config
		tempService := &EmbeddingService{
			baseURL:   strings.TrimSuffix(config.BaseURL, "/"),
			apiKey:    config.APIKey,
			model:     "text-embedding-3-small", // Default embedding model
			dimension: models.DimensionDefault,
			client:    s.client,
		}
		return tempService.EmbedBatch(ctx, texts)

	case models.ProviderTypeAnthropic:
		// Anthropic doesn't have a public embedding API yet
		// Fall back to default service
		log.Printf("[Embedding] Anthropic doesn't support embeddings, falling back to default")
		return s.EmbedBatch(ctx, texts)

	case models.ProviderTypeGoogle:
		// Google has different embedding endpoint format
		return s.embedWithGoogle(ctx, texts, config)

	default:
		return s.EmbedBatch(ctx, texts)
	}
}

// embedWithGoogle handles Google's embedding API format
func (s *EmbeddingService) embedWithGoogle(ctx context.Context, texts []string, config *AIProviderConfig) ([][]float32, error) {
	// Google uses a different request format
	type googleEmbedRequest struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	}

	type googleEmbedResponse struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	embeddings := make([][]float32, len(texts))

	// Google API processes one text at a time
	for i, text := range texts {
		if len(text) > 8000 {
			text = text[:8000]
		}

		reqBody := googleEmbedRequest{}
		reqBody.Content.Parts = []struct {
			Text string `json:"text"`
		}{{Text: text}}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		// Google uses models/text-embedding-004 or similar
		model := "text-embedding-004"
		url := fmt.Sprintf("%s/models/%s:embedContent?key=%s",
			strings.TrimSuffix(config.BaseURL, "/"),
			model,
			config.APIKey,
		)

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Google embedding API error: %s - %s", resp.Status, string(body))
		}

		var embedResp googleEmbedResponse
		if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
			return nil, err
		}

		embeddings[i] = embedResp.Embedding.Values
	}

	return embeddings, nil
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

// ChunkText splits long text into chunks for embedding
// Returns chunks that can be individually embedded
func ChunkText(text string, maxChunkSize int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 1000 // Default chunk size in characters
	}

	if len(text) <= maxChunkSize {
		return []string{text}
	}

	var chunks []string
	sentences := strings.Split(text, ". ")

	var currentChunk strings.Builder
	for _, sentence := range sentences {
		if currentChunk.Len()+len(sentence)+2 > maxChunkSize {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
		}
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(". ")
		}
		currentChunk.WriteString(sentence)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}
