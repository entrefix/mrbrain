package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

type MemoryService struct {
	memoryRepo        *repository.MemoryRepository
	todoRepo          *repository.TodoRepository
	aiService         *AIService
	aiProviderService *AIProviderService
	scraperService    *ScraperService
	ragService        *RAGService
	cacheService      *CacheService
}

func NewMemoryService(
	memoryRepo *repository.MemoryRepository,
	todoRepo *repository.TodoRepository,
	aiService *AIService,
	aiProviderService *AIProviderService,
	scraperService *ScraperService,
	ragService *RAGService,
	cacheService *CacheService,
) *MemoryService {
	return &MemoryService{
		memoryRepo:        memoryRepo,
		todoRepo:          todoRepo,
		aiService:         aiService,
		aiProviderService: aiProviderService,
		scraperService:    scraperService,
		ragService:        ragService,
		cacheService:      cacheService,
	}
}

// Create processes and stores a new memory using 2-step AI function calling
func (s *MemoryService) Create(userID string, req *models.MemoryCreateRequest) (*models.Memory, error) {
	log.Printf("[MemoryService] Creating memory for user %s: %q", userID, req.Content)

	// Get max position for new memory
	maxPos, err := s.memoryRepo.GetMaxPosition(userID)
	if err != nil {
		maxPos = 0
	}

	memory := &models.Memory{
		UserID:   userID,
		Content:  req.Content,
		Category: "Uncategorized",
		Position: fmt.Sprintf("%d", maxPos+1000),
	}

	// Get AI config
	config := s.getAIConfig(userID)

	// Use function calling for 2-step AI processing
	// Step 1: AI categorizes and detects URLs
	// Step 2: If URL detected, AI scrapes and summarizes
	if config != nil {
		memoryResult, urlSummary, err := ProcessMemoryWithFunctionCalling(
			req.Content,
			config,
			s.scraperService,
		)

		if err == nil && memoryResult != nil {
			memory.Category = memoryResult.Category
			if memoryResult.Summary != "" {
				memory.Summary = &memoryResult.Summary
			}
		}

		// Apply URL summary if we got one from the 2-step process
		if urlSummary != nil {
			// Extract URL from content for storage
			detectedURL := ExtractURLFromText(req.Content)
			if detectedURL != nil {
				memory.URL = detectedURL
			}
			if urlSummary.Title != "" {
				memory.URLTitle = &urlSummary.Title
			}
			if urlSummary.Summary != "" {
				memory.URLContent = &urlSummary.Summary
			}
		} else {
			// Fallback: Check for URL manually if function calling didn't detect one
			detectedURL := ExtractURLFromText(req.Content)
			if detectedURL != nil {
				memory.URL = detectedURL
				log.Printf("[MemoryService] Fallback URL detection: %s", *detectedURL)

				// Try to scrape if we have a scraper
				if s.scraperService != nil {
					scraped, err := s.scraperService.ScrapeURL(*detectedURL)
					if err == nil && scraped != nil {
						memory.URLTitle = &scraped.Title
						if config != nil && scraped.Content != "" {
							urlSummaryResult, _ := SummarizeURLWithProvider(*detectedURL, scraped.Content, config)
							if urlSummaryResult != nil {
								if urlSummaryResult.Title != "" {
									memory.URLTitle = &urlSummaryResult.Title
								}
								if urlSummaryResult.Summary != "" {
									memory.URLContent = &urlSummaryResult.Summary
								}
							}
						}
					}
				}
			}
		}
	} else {
		// No AI config - just detect URL manually
		detectedURL := ExtractURLFromText(req.Content)
		if detectedURL != nil {
			memory.URL = detectedURL
		}
	}

	// Store memory
	if err := s.memoryRepo.Create(memory); err != nil {
		return nil, err
	}

	// Async RAG indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		log.Printf("[MemoryService] Indexing memory %s to vector database (async)", memory.ID)
		go func(m *models.Memory) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexMemory(ctx, m); err != nil {
				log.Printf("[MemoryService] Failed to index memory %s: %v", m.ID, err)
			} else {
				log.Printf("[MemoryService] Successfully indexed memory %s", m.ID)
			}
		}(memory)
	}

	log.Printf("[MemoryService] Created memory %s with category %s", memory.ID, memory.Category)
	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserMemories(userID)
	}

	return memory, nil
}

// CreateFromChat creates a memory from chat interface (simplified flow, content already processed)
func (s *MemoryService) CreateFromChat(userID string, req *models.MemoryCreateFromChatRequest) (*models.Memory, error) {
	log.Printf("[MemoryService] Creating memory from chat for user %s: %q", userID, req.Content)

	// Get max position for new memory
	maxPos, err := s.memoryRepo.GetMaxPosition(userID)
	if err != nil {
		maxPos = 0
	}

	memory := &models.Memory{
		UserID:   userID,
		Content:  req.Content,
		Category: "Uncategorized",
		Position: fmt.Sprintf("%d", maxPos+1000),
	}

	// Use provided category if available, otherwise use AI categorization
	if req.Category != nil && *req.Category != "" {
		memory.Category = *req.Category
	} else {
		// Try AI categorization if available
		config := s.getAIConfig(userID)
		if config != nil {
			memoryResult, _, err := ProcessMemoryWithFunctionCalling(
				req.Content,
				config,
				s.scraperService,
			)
			if err == nil && memoryResult != nil {
				memory.Category = memoryResult.Category
				if memoryResult.Summary != "" {
					memory.Summary = &memoryResult.Summary
				}
			}
		}
	}

	// Use provided summary if available
	if req.Summary != nil && *req.Summary != "" {
		memory.Summary = req.Summary
	}

	// Store memory
	if err := s.memoryRepo.Create(memory); err != nil {
		return nil, err
	}

	// Async RAG indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		log.Printf("[MemoryService] Indexing memory %s to vector database (async)", memory.ID)
		go func(m *models.Memory) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexMemory(ctx, m); err != nil {
				log.Printf("[MemoryService] Failed to index memory %s: %v", m.ID, err)
			} else {
				log.Printf("[MemoryService] Successfully indexed memory %s", m.ID)
			}
		}(memory)
	}

	log.Printf("[MemoryService] Created memory from chat %s with category %s", memory.ID, memory.Category)
	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserMemories(userID)
	}

	return memory, nil
}

// CreateWithCategory creates a memory with pre-determined category and summary (used by vision service)
func (s *MemoryService) CreateWithCategory(userID string, req *models.MemoryCreateRequest, category, summary string) (*models.Memory, error) {
	log.Printf("[MemoryService] Creating memory with category for user %s: category=%s", userID, category)

	// Get max position for new memory
	maxPos, err := s.memoryRepo.GetMaxPosition(userID)
	if err != nil {
		maxPos = 0
	}

	memory := &models.Memory{
		UserID:   userID,
		Content:  req.Content,
		Category: category,
		Position: fmt.Sprintf("%d", maxPos+1000),
	}

	if summary != "" {
		memory.Summary = &summary
	}

	// Store memory
	if err := s.memoryRepo.Create(memory); err != nil {
		return nil, err
	}

	// Async RAG indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		log.Printf("[MemoryService] Indexing memory %s to vector database (async)", memory.ID)
		go func(m *models.Memory) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexMemory(ctx, m); err != nil {
				log.Printf("[MemoryService] Failed to index memory %s: %v", m.ID, err)
			} else {
				log.Printf("[MemoryService] Successfully indexed memory %s", m.ID)
			}
		}(memory)
	}

	log.Printf("[MemoryService] Created memory %s with category %s (from vision)", memory.ID, memory.Category)
	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserMemories(userID)
	}

	return memory, nil
}

// getAIConfig returns the AI provider configuration for a user
func (s *MemoryService) getAIConfig(userID string) *AIProviderConfig {
	// Try user's configured provider first
	if s.aiProviderService != nil {
		provider, err := s.aiProviderService.GetDefaultByUserID(userID)
		if err == nil && provider != nil && provider.SelectedModel != nil {
			apiKey, err := s.aiProviderService.GetDecryptedAPIKey(provider)
			if err == nil {
				return &AIProviderConfig{
					ProviderType: provider.ProviderType,
					BaseURL:      provider.BaseURL,
					APIKey:       apiKey,
					Model:        *provider.SelectedModel,
				}
			}
		}
	}

	// Fall back to default AI service
	if s.aiService != nil && s.aiService.IsConfigured() {
		return &AIProviderConfig{
			ProviderType: models.ProviderTypeOpenAI,
			BaseURL:      s.aiService.baseURL,
			APIKey:       s.aiService.apiKey,
			Model:        s.aiService.model,
		}
	}

	return nil
}

// GetAll retrieves memories with pagination
func (s *MemoryService) GetAll(userID string, limit, offset int) ([]models.Memory, error) {
	// Note: Caching with limit/offset is complex, so we only cache when limit/offset are default values
	// For paginated requests, skip cache
	if limit != 0 || offset != 0 {
		return s.memoryRepo.GetAllByUserID(userID, limit, offset)
	}

	// Try to get from cache first (only for non-paginated requests)
	if s.cacheService != nil {
		cached, err := s.cacheService.GetCachedUserMemories(userID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Cache miss or Redis unavailable - fetch from database
	memories, err := s.memoryRepo.GetAllByUserID(userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache the result (async, don't block) - only for non-paginated
	if s.cacheService != nil && limit == 0 && offset == 0 {
		go func() {
			_ = s.cacheService.CacheUserMemories(userID, memories)
		}()
	}

	return memories, nil
}

// GetByID retrieves a single memory
func (s *MemoryService) GetByID(userID, memoryID string) (*models.Memory, error) {
	memory, err := s.memoryRepo.GetByID(memoryID)
	if err != nil {
		return nil, err
	}
	if memory == nil || memory.UserID != userID {
		return nil, nil
	}
	return memory, nil
}

// GetByCategory retrieves memories filtered by category
func (s *MemoryService) GetByCategory(userID, category string, limit, offset int) ([]models.Memory, error) {
	return s.memoryRepo.GetByCategory(userID, category, limit, offset)
}

// Search performs full-text search
func (s *MemoryService) Search(userID string, req *models.MemorySearchRequest) ([]models.Memory, error) {
	return s.memoryRepo.Search(userID, req)
}

// Update updates a memory
func (s *MemoryService) Update(userID, memoryID string, req *models.MemoryUpdateRequest) (*models.Memory, error) {
	// Verify ownership
	memory, err := s.memoryRepo.GetByID(memoryID)
	if err != nil {
		return nil, err
	}
	if memory == nil || memory.UserID != userID {
		return nil, fmt.Errorf("memory not found")
	}

	updates := make(map[string]interface{})

	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.IsArchived != nil {
		if *req.IsArchived {
			updates["is_archived"] = 1
		} else {
			updates["is_archived"] = 0
		}
	}

	if len(updates) > 0 {
		if err := s.memoryRepo.Update(memoryID, updates); err != nil {
			return nil, err
		}
	}

	updatedMemory, err := s.memoryRepo.GetByID(memoryID)
	if err != nil {
		return nil, err
	}

	// Async RAG re-indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() && updatedMemory != nil {
		log.Printf("[MemoryService] Re-indexing updated memory %s to vector database (async)", updatedMemory.ID)
		go func(m *models.Memory) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexMemory(ctx, m); err != nil {
				log.Printf("[MemoryService] Failed to re-index memory %s: %v", m.ID, err)
			} else {
				log.Printf("[MemoryService] Successfully re-indexed memory %s", m.ID)
			}
		}(updatedMemory)
	}

	return updatedMemory, nil
}

// Delete removes a memory
func (s *MemoryService) Delete(userID, memoryID string) error {
	// Verify ownership
	memory, err := s.memoryRepo.GetByID(memoryID)
	if err != nil {
		return err
	}
	if memory == nil || memory.UserID != userID {
		return fmt.Errorf("memory not found")
	}

	// Delete from RAG indexes FIRST (synchronously for reliability)
	if s.ragService != nil && s.ragService.IsConfigured() {
		log.Printf("[MemoryService] Deleting memory %s from vector and FTS indexes", memoryID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.ragService.DeleteFromIndex(ctx, models.ContentTypeMemory, memoryID); err != nil {
			log.Printf("[MemoryService] Warning: Failed to delete memory %s from indexes: %v", memoryID, err)
			// Don't fail - continue with database deletion to prevent orphaned records
		}
	}

	// Delete from database (FTS will be auto-deleted by SQLite trigger)
	return s.memoryRepo.Delete(memoryID)
}

// ConvertToTodo creates a todo from a memory
func (s *MemoryService) ConvertToTodo(userID, memoryID string, req *models.MemoryToTodoRequest) (*models.Todo, error) {
	// Get the memory
	memory, err := s.memoryRepo.GetByID(memoryID)
	if err != nil {
		return nil, err
	}
	if memory == nil || memory.UserID != userID {
		return nil, fmt.Errorf("memory not found")
	}

	// Determine title and description
	title := memory.Content
	if req.Title != nil && *req.Title != "" {
		title = *req.Title
	} else if memory.Summary != nil && *memory.Summary != "" {
		title = *memory.Summary
	}

	// Truncate title if too long
	if len(title) > 200 {
		title = title[:200] + "..."
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	} else if memory.Summary != nil {
		description = &memory.Content
	}

	// Determine priority
	priority := models.PriorityMedium
	if req.Priority != nil {
		priority = models.Priority(*req.Priority)
	}

	// Get max position
	maxPos, err := s.todoRepo.GetMaxPosition(userID)
	if err != nil {
		maxPos = 0
	}

	// Create the todo
	todo := &models.Todo{
		UserID:      userID,
		GroupID:     req.GroupID,
		Title:       title,
		Description: description,
		Priority:    priority,
		Position:    fmt.Sprintf("%d", maxPos+1000),
		Tags:        []string{"from-memory"},
	}

	if err := s.todoRepo.Create(todo); err != nil {
		return nil, err
	}

	// Async RAG indexing for the new todo - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		go func(t *models.Todo) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexTodo(ctx, t); err != nil {
				log.Printf("[MemoryService] Failed to index converted todo %s: %v", t.ID, err)
			}
		}(todo)
	}

	return todo, nil
}

// GetCategories returns all available categories
func (s *MemoryService) GetCategories(userID string) ([]models.MemoryCategory, error) {
	return s.memoryRepo.GetCategories(userID)
}

// Reorder updates positions for multiple memories
func (s *MemoryService) Reorder(userID string, req *models.MemoryReorderRequest) error {
	// Verify all memories belong to user before updating
	for _, m := range req.Memories {
		memory, err := s.memoryRepo.GetByID(m.ID)
		if err != nil {
			return err
		}
		if memory == nil || memory.UserID != userID {
			return fmt.Errorf("memory %s not found or unauthorized", m.ID)
		}
	}

	return s.memoryRepo.UpdatePositions(req.Memories)
}

// GetStats returns memory statistics
func (s *MemoryService) GetStats(userID string) (*models.MemoryStats, error) {
	return s.memoryRepo.GetStats(userID)
}

// GetOrGenerateDigest retrieves or creates weekly digest
func (s *MemoryService) GetOrGenerateDigest(userID string, forceRegenerate bool) (*models.MemoryDigest, error) {
	// Calculate current week start (Sunday)
	now := time.Now()
	weekday := int(now.Weekday())
	weekStart := now.AddDate(0, 0, -weekday)
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	weekEnd := weekStart.AddDate(0, 0, 6)

	// Check if digest already exists
	if !forceRegenerate {
		existing, err := s.memoryRepo.GetDigest(userID, weekStart)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	// Get memories from this week
	memories, err := s.memoryRepo.GetByDateRange(userID, weekStart, weekEnd.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}

	// Generate digest with AI
	config := s.getAIConfig(userID)
	if config == nil {
		return nil, fmt.Errorf("AI not configured")
	}

	digestContent, err := GenerateWeeklyDigestWithProvider(memories, config)
	if err != nil {
		return nil, err
	}

	// Save digest
	digest := &models.MemoryDigest{
		UserID:        userID,
		WeekStart:     weekStart.Format("2006-01-02"),
		WeekEnd:       weekEnd.Format("2006-01-02"),
		DigestContent: digestContent,
	}

	if err := s.memoryRepo.SaveDigest(digest); err != nil {
		return nil, err
	}

	return digest, nil
}

// WebSearch searches the web using SearXNG
func (s *MemoryService) WebSearch(query string) ([]models.WebSearchResult, error) {
	if s.scraperService == nil {
		return nil, fmt.Errorf("web search not configured")
	}

	results, err := s.scraperService.SearchWeb(query)
	if err != nil {
		return nil, err
	}

	webResults := make([]models.WebSearchResult, len(results))
	for i, r := range results {
		webResults[i] = models.WebSearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
		}
	}

	return webResults, nil
}
