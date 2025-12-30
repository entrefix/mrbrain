package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

// RAGService provides Retrieval-Augmented Generation capabilities
type RAGService struct {
	vectorRepo       *repository.VectorRepository
	ftsRepo          *repository.FTSRepository
	todoRepo         *repository.TodoRepository
	memoryRepo       *repository.MemoryRepository
	embeddingService *EmbeddingService
	aiService        *AIService
	aiProviderSvc    *AIProviderService
}

// RAGConfig holds configuration for the RAG service
type RAGConfig struct {
	VectorPersistPath  string
	EmbeddingDimension int
}

// NewRAGService creates a new RAG service
func NewRAGService(
	vectorRepo *repository.VectorRepository,
	ftsRepo *repository.FTSRepository,
	todoRepo *repository.TodoRepository,
	memoryRepo *repository.MemoryRepository,
	embeddingService *EmbeddingService,
	aiService *AIService,
	aiProviderSvc *AIProviderService,
) *RAGService {
	return &RAGService{
		vectorRepo:       vectorRepo,
		ftsRepo:          ftsRepo,
		todoRepo:         todoRepo,
		memoryRepo:       memoryRepo,
		embeddingService: embeddingService,
		aiService:        aiService,
		aiProviderSvc:    aiProviderSvc,
	}
}

// IsConfigured returns true if RAG service is properly configured
func (s *RAGService) IsConfigured() bool {
	return s.embeddingService != nil && s.embeddingService.IsConfigured() && s.vectorRepo != nil
}

// ==========================================
// Hybrid Search
// ==========================================

// Search performs hybrid search combining vector similarity and keyword matching
func (s *RAGService) Search(ctx context.Context, userID string, req *models.SearchRequest) (*models.SearchResponse, error) {
	startTime := time.Now()

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.VectorWeight <= 0 {
		req.VectorWeight = 0.7 // Default: favor vector search
	}

	log.Printf("[RAG] Hybrid search: user=%s, query=%q, limit=%d, vector_weight=%.2f",
		userID, req.Query, req.Limit, req.VectorWeight)

	var vectorResults, keywordResults []models.SearchResult
	var vecErr, ftsErr error

	// Run vector and keyword search in parallel
	done := make(chan bool, 2)

	// Vector search
	go func() {
		if s.vectorRepo != nil && s.embeddingService.IsConfigured() {
			vectorResults, vecErr = s.vectorRepo.SearchByUser(ctx, userID, req.Query, req.Limit*2, req.ContentTypes)
		}
		done <- true
	}()

	// Keyword search
	go func() {
		if s.ftsRepo != nil {
			keywordResults, ftsErr = s.ftsRepo.SearchWithHighlights(userID, req.Query, req.ContentTypes, req.Limit*2)
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	if vecErr != nil {
		log.Printf("[RAG] Vector search error: %v", vecErr)
	}
	if ftsErr != nil {
		log.Printf("[RAG] Keyword search error: %v", ftsErr)
	}

	// Combine results using Reciprocal Rank Fusion
	combined := s.reciprocalRankFusion(vectorResults, keywordResults, req.VectorWeight)

	// Limit results
	if len(combined) > req.Limit {
		combined = combined[:req.Limit]
	}

	// Enrich results with full document data
	enriched := s.enrichSearchResults(ctx, userID, combined)

	return &models.SearchResponse{
		Results:    enriched,
		Query:      req.Query,
		TotalCount: len(enriched),
		TimeTaken:  float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// reciprocalRankFusion combines results from multiple search methods
func (s *RAGService) reciprocalRankFusion(vectorResults, keywordResults []models.SearchResult, vectorWeight float64) []models.SearchResult {
	const k = 60.0 // RRF constant

	// Map to track combined scores by content_id
	scoreMap := make(map[string]float64)
	docMap := make(map[string]*models.SearchResult)

	// Add vector results
	for i, result := range vectorResults {
		key := fmt.Sprintf("%s-%s", result.Document.ContentType, result.Document.ContentID)
		rrf := vectorWeight * (1.0 / (k + float64(i+1)))
		scoreMap[key] += rrf
		if _, exists := docMap[key]; !exists {
			r := result
			r.MatchType = "vector"
			docMap[key] = &r
		}
	}

	// Add keyword results
	keywordWeight := 1.0 - vectorWeight
	for i, result := range keywordResults {
		key := fmt.Sprintf("%s-%s", result.Document.ContentType, result.Document.ContentID)
		rrf := keywordWeight * (1.0 / (k + float64(i+1)))
		scoreMap[key] += rrf

		if existing, exists := docMap[key]; exists {
			existing.MatchType = "hybrid"
			// Merge highlights
			if len(result.Highlights) > 0 {
				existing.Highlights = append(existing.Highlights, result.Highlights...)
			}
		} else {
			r := result
			r.MatchType = "keyword"
			docMap[key] = &r
		}
	}

	// Convert to slice and sort by combined score
	var combined []models.SearchResult
	for key, result := range docMap {
		result.Score = scoreMap[key]
		combined = append(combined, *result)
	}

	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Score > combined[j].Score
	})

	return combined
}

// enrichSearchResults adds full document data to search results
func (s *RAGService) enrichSearchResults(ctx context.Context, userID string, results []models.SearchResult) []models.SearchResult {
	enriched := make([]models.SearchResult, 0, len(results))

	for _, result := range results {
		switch result.Document.ContentType {
		case models.ContentTypeTodo:
			if todo, _ := s.todoRepo.GetByID(result.Document.ContentID); todo != nil {
				result.Document.Title = todo.Title
				if todo.Description != nil {
					result.Document.Content = *todo.Description
				}
				result.Document.Metadata = map[string]string{
					"priority": string(todo.Priority),
					"status":   string(todo.Status),
				}
				if todo.GroupID != nil {
					result.Document.Metadata["group_id"] = *todo.GroupID
				}
				if len(todo.Tags) > 0 {
					tagsJSON, _ := json.Marshal(todo.Tags)
					result.Document.Metadata["tags"] = string(tagsJSON)
				}
				if todo.DueDate != nil {
					result.Document.Metadata["due_date"] = *todo.DueDate
				}
			}

		case models.ContentTypeMemory:
			if memory, _ := s.memoryRepo.GetByID(result.Document.ContentID); memory != nil {
				result.Document.Content = memory.Content
				if memory.URLTitle != nil {
					result.Document.Title = *memory.URLTitle
				}
				result.Document.Metadata = map[string]string{
					"category": memory.Category,
				}
				if memory.Summary != nil {
					result.Document.Metadata["summary"] = *memory.Summary
				}
				if memory.URL != nil {
					result.Document.Metadata["url"] = *memory.URL
				}
			}
		}

		enriched = append(enriched, result)
	}

	return enriched
}

// ==========================================
// Q&A (Ask)
// ==========================================

// Ask answers a question using RAG
func (s *RAGService) Ask(ctx context.Context, userID string, req *models.AskRequest) (*models.AskResponse, error) {
	startTime := time.Now()

	if req.MaxContext <= 0 {
		req.MaxContext = 5
	}

	log.Printf("[RAG] Ask: user=%s, question=%q", userID, req.Question)

	// First, search for relevant documents
	searchReq := &models.SearchRequest{
		Query:        req.Question,
		ContentTypes: req.ContentTypes,
		Limit:        req.MaxContext,
		VectorWeight: 0.7,
	}

	searchResp, err := s.Search(ctx, userID, searchReq)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResp.Results) == 0 {
		return &models.AskResponse{
			Answer:    "I couldn't find any relevant information to answer your question.",
			Sources:   []models.SearchResult{},
			Question:  req.Question,
			TimeTaken: float64(time.Since(startTime).Milliseconds()),
		}, nil
	}

	// Build context from search results
	contextParts := make([]string, 0, len(searchResp.Results))
	for i, result := range searchResp.Results {
		var contextItem string
		switch result.Document.ContentType {
		case models.ContentTypeTodo:
			contextItem = fmt.Sprintf("[Todo %d] %s", i+1, result.Document.Title)
			if result.Document.Content != "" {
				contextItem += "\n  Description: " + result.Document.Content
			}
			if status, ok := result.Document.Metadata["status"]; ok {
				contextItem += "\n  Status: " + status
			}
			if dueDate, ok := result.Document.Metadata["due_date"]; ok {
				contextItem += "\n  Due: " + dueDate
			}

		case models.ContentTypeMemory:
			contextItem = fmt.Sprintf("[Memory %d] %s", i+1, result.Document.Content)
			if result.Document.Title != "" {
				contextItem = fmt.Sprintf("[Memory %d - %s] %s", i+1, result.Document.Title, result.Document.Content)
			}
			if category, ok := result.Document.Metadata["category"]; ok {
				contextItem += "\n  Category: " + category
			}
			if summary, ok := result.Document.Metadata["summary"]; ok && summary != "" {
				contextItem += "\n  Summary: " + summary
			}
		}
		contextParts = append(contextParts, contextItem)
	}

	contextStr := strings.Join(contextParts, "\n\n")

	// Generate answer using AI
	answer, err := s.generateAnswer(ctx, userID, req.Question, contextStr)
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &models.AskResponse{
		Answer:    answer,
		Sources:   searchResp.Results,
		Question:  req.Question,
		TimeTaken: float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// generateAnswer uses AI to answer the question based on context
func (s *RAGService) generateAnswer(ctx context.Context, userID, question, contextStr string) (string, error) {
	prompt := fmt.Sprintf(`You are a helpful assistant answering questions about a user's personal data (todos and memories).

Based on the following context from the user's data, answer their question concisely and helpfully.
If the context doesn't contain relevant information, say so clearly.

CONTEXT:
%s

QUESTION: %s

INSTRUCTIONS:
- Answer based ONLY on the provided context
- Be concise but complete
- If referencing specific items, mention them clearly
- If the answer isn't in the context, say "I don't have enough information to answer that"
- Don't make up information not present in the context

ANSWER:`, contextStr, question)

	// Try to use user's configured AI provider first
	if s.aiProviderSvc != nil {
		provider, err := s.aiProviderSvc.GetDefaultByUserID(userID)
		if err == nil && provider != nil {
			apiKey, err := s.aiProviderSvc.GetDecryptedAPIKey(provider)
			if err == nil {
				model := os.Getenv("OPENAI_MODEL")
				if provider.SelectedModel != nil {
					model = *provider.SelectedModel
				}
				config := &AIProviderConfig{
					ProviderType: provider.ProviderType,
					BaseURL:      provider.BaseURL,
					APIKey:       apiKey,
					Model:        model,
				}
				switch config.ProviderType {
				case models.ProviderTypeAnthropic:
					return callAnthropic(config, prompt)
				case models.ProviderTypeGoogle:
					return callGoogle(config, prompt)
				default:
					return callOpenAICompatible(config, prompt)
				}
			}
		}
	}

	// Fall back to default AI service
	if s.aiService != nil && s.aiService.IsConfigured() {
		config := &AIProviderConfig{
			ProviderType: models.ProviderTypeOpenAI,
			BaseURL:      s.aiService.baseURL,
			APIKey:       s.aiService.apiKey,
			Model:        s.aiService.model,
		}
		return callOpenAICompatible(config, prompt)
	}

	return "", fmt.Errorf("no AI service configured")
}

// ==========================================
// Indexing
// ==========================================

// IndexAllForUser indexes all todos and memories for a user
func (s *RAGService) IndexAllForUser(ctx context.Context, userID string) (*models.IndexResponse, error) {
	startTime := time.Now()
	var indexed, skipped, errors int

	log.Printf("[RAG] Starting full index for user: %s", userID)

	// Index todos
	todos, err := s.todoRepo.GetAllByUserID(userID)
	if err != nil {
		log.Printf("[RAG] Error fetching todos: %v", err)
	} else {
		for _, todo := range todos {
			// Check if already indexed
			if s.vectorRepo.GetByContentID(models.ContentTypeTodo, todo.ID) != nil {
				skipped++
				continue
			}

			doc := s.todoToDocument(&todo)
			if err := s.vectorRepo.Add(ctx, doc); err != nil {
				log.Printf("[RAG] Error indexing todo %s: %v", todo.ID, err)
				errors++
			} else {
				indexed++
			}
		}
	}

	// Index memories
	memories, err := s.memoryRepo.GetAllByUserID(userID, 1000, 0)
	if err != nil {
		log.Printf("[RAG] Error fetching memories: %v", err)
	} else {
		for _, memory := range memories {
			// Check if already indexed
			if s.vectorRepo.GetByContentID(models.ContentTypeMemory, memory.ID) != nil {
				skipped++
				continue
			}

			doc := s.memoryToDocument(&memory)
			if err := s.vectorRepo.Add(ctx, doc); err != nil {
				log.Printf("[RAG] Error indexing memory %s: %v", memory.ID, err)
				errors++
			} else {
				indexed++
			}
		}
	}

	log.Printf("[RAG] Indexing complete: indexed=%d, skipped=%d, errors=%d", indexed, skipped, errors)

	return &models.IndexResponse{
		Indexed:   indexed,
		Skipped:   skipped,
		Errors:    errors,
		TimeTaken: float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// IndexTodo indexes a single todo
func (s *RAGService) IndexTodo(ctx context.Context, todo *models.Todo) error {
	if !s.IsConfigured() {
		return nil // Silently skip if not configured
	}

	// Delete existing if present
	s.vectorRepo.DeleteByContentID(ctx, models.ContentTypeTodo, todo.ID)

	doc := s.todoToDocument(todo)
	return s.vectorRepo.Add(ctx, doc)
}

// IndexMemory indexes a single memory
func (s *RAGService) IndexMemory(ctx context.Context, memory *models.Memory) error {
	if !s.IsConfigured() {
		return nil // Silently skip if not configured
	}

	// Delete existing if present
	s.vectorRepo.DeleteByContentID(ctx, models.ContentTypeMemory, memory.ID)

	doc := s.memoryToDocument(memory)
	return s.vectorRepo.Add(ctx, doc)
}

// DeleteFromIndex removes a document from the index
func (s *RAGService) DeleteFromIndex(ctx context.Context, contentType models.ContentType, contentID string) error {
	if !s.IsConfigured() {
		return nil
	}
	return s.vectorRepo.DeleteByContentID(ctx, contentType, contentID)
}

// ==========================================
// Helpers
// ==========================================

func (s *RAGService) todoToDocument(todo *models.Todo) *models.Document {
	content := todo.Title
	if todo.Description != nil && *todo.Description != "" {
		content += "\n" + *todo.Description
	}

	metadata := map[string]string{
		"priority": string(todo.Priority),
		"status":   string(todo.Status),
	}

	if todo.GroupID != nil {
		metadata["group_id"] = *todo.GroupID
	}

	if len(todo.Tags) > 0 {
		tagsJSON, _ := json.Marshal(todo.Tags)
		metadata["tags"] = string(tagsJSON)
	}

	if todo.DueDate != nil {
		metadata["due_date"] = *todo.DueDate
	}

	return &models.Document{
		ContentType: models.ContentTypeTodo,
		ContentID:   todo.ID,
		UserID:      todo.UserID,
		Title:       todo.Title,
		Content:     content,
		Metadata:    metadata,
		CreatedAt:   todo.CreatedAt,
	}
}

func (s *RAGService) memoryToDocument(memory *models.Memory) *models.Document {
	title := ""
	if memory.URLTitle != nil {
		title = *memory.URLTitle
	}

	content := memory.Content
	if memory.Summary != nil && *memory.Summary != "" {
		content += "\nSummary: " + *memory.Summary
	}

	metadata := map[string]string{
		"category": memory.Category,
	}

	if memory.URL != nil {
		metadata["url"] = *memory.URL
	}

	return &models.Document{
		ContentType: models.ContentTypeMemory,
		ContentID:   memory.ID,
		UserID:      memory.UserID,
		Title:       title,
		Content:     content,
		Metadata:    metadata,
		CreatedAt:   memory.CreatedAt,
	}
}

// GetStats returns RAG index statistics
func (s *RAGService) GetStats(userID string) *models.IndexStats {
	if s.vectorRepo == nil {
		return &models.IndexStats{
			TotalDocuments: 0,
			ByContentType:  make(map[string]int),
			ByUser:         make(map[string]int),
		}
	}
	return s.vectorRepo.GetStats(userID)
}
