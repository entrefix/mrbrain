package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

// RAGService provides Retrieval-Augmented Generation capabilities using ClaraVector
type RAGService struct {
	claraClient   *ClaraVectorClient
	todoRepo      *repository.TodoRepository
	memoryRepo    *repository.MemoryRepository
	aiService     *AIService
	aiProviderSvc *AIProviderService
}

// NewRAGService creates a new RAG service
func NewRAGService(
	claraClient *ClaraVectorClient,
	todoRepo *repository.TodoRepository,
	memoryRepo *repository.MemoryRepository,
	aiService *AIService,
	aiProviderSvc *AIProviderService,
) *RAGService {
	return &RAGService{
		claraClient:   claraClient,
		todoRepo:      todoRepo,
		memoryRepo:    memoryRepo,
		aiService:     aiService,
		aiProviderSvc: aiProviderSvc,
	}
}

// IsConfigured returns true if RAG service is properly configured
func (s *RAGService) IsConfigured() bool {
	return s.claraClient != nil && s.claraClient.IsConfigured()
}

// ==========================================
// Search
// ==========================================

// Search performs semantic search using ClaraVector
func (s *RAGService) Search(ctx context.Context, userID string, req *models.SearchRequest) (*models.SearchResponse, error) {
	startTime := time.Now()

	if req.Limit <= 0 {
		req.Limit = 10
	}

	log.Printf("[RAG] Search: user=%s, query=%q, limit=%d", userID, req.Query, req.Limit)

	// Parse userID to uint
	userIDUint, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Query ClaraVector
	results, err := s.claraClient.QueryUser(uint(userIDUint), req.Query, req.Limit)
	if err != nil {
		log.Printf("[RAG] ClaraVector query error: %v", err)
		return &models.SearchResponse{
			Results:    []models.SearchResult{},
			Query:      req.Query,
			TotalCount: 0,
			TimeTaken:  float64(time.Since(startTime).Milliseconds()),
		}, nil
	}

	if len(results) == 0 {
		return &models.SearchResponse{
			Results:    []models.SearchResult{},
			Query:      req.Query,
			TotalCount: 0,
			TimeTaken:  float64(time.Since(startTime).Milliseconds()),
		}, nil
	}

	// Convert ClaraVector results to our format and enrich
	searchResults := s.convertAndEnrichResults(ctx, userID, results, req.ContentTypes)

	return &models.SearchResponse{
		Results:    searchResults,
		Query:      req.Query,
		TotalCount: len(searchResults),
		TimeTaken:  float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// convertAndEnrichResults converts ClaraVector results and enriches with local data
func (s *RAGService) convertAndEnrichResults(ctx context.Context, userID string, results []ClaraVectorQueryResult, contentTypes []string) []models.SearchResult {
	searchResults := make([]models.SearchResult, 0, len(results))

	// Get all todos and memories for the user to match against
	todos, _ := s.todoRepo.GetAllByUserID(userID)
	memories, _ := s.memoryRepo.GetAllByUserID(userID, 1000, 0)

	for _, result := range results {
		// Try to match the result text to a todo or memory
		var matched bool

		// Check if we should filter by content type
		filterTodos := len(contentTypes) == 0 || containsString(contentTypes, "todo")
		filterMemories := len(contentTypes) == 0 || containsString(contentTypes, "memory")

		// Try matching against todos
		if filterTodos {
			for _, todo := range todos {
				content := todo.Title
				if todo.Description != nil {
					content += "\n" + *todo.Description
				}
				if strings.Contains(content, result.Text) || strings.Contains(result.Text, todo.Title) {
					doc := &models.Document{
						ContentType: models.ContentTypeTodo,
						ContentID:   todo.ID,
						UserID:      todo.UserID,
						Title:       todo.Title,
						Metadata:    make(map[string]string),
					}
					if todo.Description != nil {
						doc.Content = *todo.Description
					}
					doc.Metadata["priority"] = string(todo.Priority)
					doc.Metadata["status"] = string(todo.Status)
					if todo.GroupID != nil {
						doc.Metadata["group_id"] = *todo.GroupID
					}
					if len(todo.Tags) > 0 {
						tagsJSON, _ := json.Marshal(todo.Tags)
						doc.Metadata["tags"] = string(tagsJSON)
					}
					if todo.DueDate != nil {
						doc.Metadata["due_date"] = *todo.DueDate
					}
					sr := models.SearchResult{
						Document:   doc,
						Score:      1.0 - result.SimilarityScore, // Lower similarity = better match
						MatchType:  "vector",
						Highlights: []string{result.Text},
					}
					searchResults = append(searchResults, sr)
					matched = true
					break
				}
			}
		}

		// Try matching against memories if not matched to todo
		if !matched && filterMemories {
			for _, memory := range memories {
				content := memory.Content
				if memory.Summary != nil {
					content += "\n" + *memory.Summary
				}
				if strings.Contains(content, result.Text) || strings.Contains(result.Text, memory.Content[:min(len(memory.Content), 100)]) {
					doc := &models.Document{
						ContentType: models.ContentTypeMemory,
						ContentID:   memory.ID,
						UserID:      memory.UserID,
						Content:     memory.Content,
						Metadata:    make(map[string]string),
					}
					if memory.URLTitle != nil {
						doc.Title = *memory.URLTitle
					}
					doc.Metadata["category"] = memory.Category
					if memory.Summary != nil {
						doc.Metadata["summary"] = *memory.Summary
					}
					if memory.URL != nil {
						doc.Metadata["url"] = *memory.URL
					}
					sr := models.SearchResult{
						Document:   doc,
						Score:      1.0 - result.SimilarityScore,
						MatchType:  "vector",
						Highlights: []string{result.Text},
					}
					searchResults = append(searchResults, sr)
					matched = true
					break
				}
			}
		}

		// If no match found, add as-is with the text from ClaraVector
		if !matched {
			doc := &models.Document{
				Content:  result.Text,
				Metadata: make(map[string]string),
			}
			sr := models.SearchResult{
				Document:   doc,
				Score:      1.0 - result.SimilarityScore,
				MatchType:  "vector",
				Highlights: []string{result.Text},
			}
			searchResults = append(searchResults, sr)
		}
	}

	return searchResults
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
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

		default:
			// For unmatched results, use the content directly
			if len(result.Highlights) > 0 {
				contextItem = fmt.Sprintf("[Result %d] %s", i+1, result.Highlights[0])
			} else {
				contextItem = fmt.Sprintf("[Result %d] %s", i+1, result.Document.Content)
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

	// Parse userID
	userIDUint, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Ensure user and notebooks exist
	todosNotebookID, err := s.claraClient.EnsureUserAndNotebook(uint(userIDUint), "todos")
	if err != nil {
		log.Printf("[RAG] Error ensuring todos notebook: %v", err)
		return nil, err
	}

	memoriesNotebookID, err := s.claraClient.EnsureUserAndNotebook(uint(userIDUint), "memories")
	if err != nil {
		log.Printf("[RAG] Error ensuring memories notebook: %v", err)
		return nil, err
	}

	// Index todos
	todos, err := s.todoRepo.GetAllByUserID(userID)
	if err != nil {
		log.Printf("[RAG] Error fetching todos: %v", err)
	} else {
		for _, todo := range todos {
			content := todo.Title
			if todo.Description != nil && *todo.Description != "" {
				content += "\n" + *todo.Description
			}

			filename := fmt.Sprintf("todo_%s", todo.ID)
			_, err := s.claraClient.UploadDocument(todosNotebookID, filename, content)
			if err != nil {
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
			content := memory.Content
			if memory.Summary != nil && *memory.Summary != "" {
				content += "\nSummary: " + *memory.Summary
			}

			filename := fmt.Sprintf("memory_%s", memory.ID)
			_, err := s.claraClient.UploadDocument(memoriesNotebookID, filename, content)
			if err != nil {
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

// IndexTodo indexes a single todo (fire-and-forget)
func (s *RAGService) IndexTodo(ctx context.Context, todo *models.Todo) error {
	if !s.IsConfigured() {
		return nil // Silently skip if not configured
	}

	// Parse userID
	userIDUint, err := strconv.ParseUint(todo.UserID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Fire-and-forget in goroutine
	go func() {
		notebookID, err := s.claraClient.EnsureUserAndNotebook(uint(userIDUint), "todos")
		if err != nil {
			log.Printf("[RAG] Error ensuring notebook for todo: %v", err)
			return
		}

		content := todo.Title
		if todo.Description != nil && *todo.Description != "" {
			content += "\n" + *todo.Description
		}

		filename := fmt.Sprintf("todo_%s", todo.ID)
		_, err = s.claraClient.UploadDocument(notebookID, filename, content)
		if err != nil {
			log.Printf("[RAG] Error indexing todo %s: %v", todo.ID, err)
		} else {
			log.Printf("[RAG] Indexed todo %s", todo.ID)
		}
	}()

	return nil
}

// IndexMemory indexes a single memory (fire-and-forget)
func (s *RAGService) IndexMemory(ctx context.Context, memory *models.Memory) error {
	if !s.IsConfigured() {
		return nil // Silently skip if not configured
	}

	// Parse userID
	userIDUint, err := strconv.ParseUint(memory.UserID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Fire-and-forget in goroutine
	go func() {
		notebookID, err := s.claraClient.EnsureUserAndNotebook(uint(userIDUint), "memories")
		if err != nil {
			log.Printf("[RAG] Error ensuring notebook for memory: %v", err)
			return
		}

		content := memory.Content
		if memory.Summary != nil && *memory.Summary != "" {
			content += "\nSummary: " + *memory.Summary
		}

		filename := fmt.Sprintf("memory_%s", memory.ID)
		_, err = s.claraClient.UploadDocument(notebookID, filename, content)
		if err != nil {
			log.Printf("[RAG] Error indexing memory %s: %v", memory.ID, err)
		} else {
			log.Printf("[RAG] Indexed memory %s", memory.ID)
		}
	}()

	return nil
}

// DeleteFromIndex removes a document from the index
// Note: ClaraVector deletion requires document ID which we don't track
// For now, this is a no-op. Future improvement: track ClaraVector doc IDs
func (s *RAGService) DeleteFromIndex(ctx context.Context, contentType models.ContentType, contentID string) error {
	if !s.IsConfigured() {
		return nil
	}
	// TODO: Implement deletion when we track ClaraVector document IDs
	log.Printf("[RAG] DeleteFromIndex called for %s:%s (not implemented yet)", contentType, contentID)
	return nil
}

// GetStats returns RAG index statistics
func (s *RAGService) GetStats(userID string) *models.IndexStats {
	// ClaraVector doesn't expose stats API easily, return basic info
	return &models.IndexStats{
		TotalDocuments: 0,
		ByContentType:  make(map[string]int),
		ByUser:         make(map[string]int),
	}
}
