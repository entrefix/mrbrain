package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

type TodoService struct {
	todoRepo          *repository.TodoRepository
	aiService         *AIService
	aiProviderService *AIProviderService
	ragService        *RAGService
	cacheService      *CacheService
}

func NewTodoService(todoRepo *repository.TodoRepository, aiService *AIService, aiProviderService *AIProviderService, ragService *RAGService, cacheService *CacheService) *TodoService {
	return &TodoService{
		todoRepo:          todoRepo,
		aiService:         aiService,
		aiProviderService: aiProviderService,
		ragService:        ragService,
		cacheService:      cacheService,
	}
}

func (s *TodoService) Create(userID string, req *models.TodoCreateRequest) (*models.Todo, error) {
	// Get max position for ordering
	maxPos, err := s.todoRepo.GetMaxPosition(userID)
	if err != nil {
		return nil, err
	}

	// Process with AI if available
	var aiResult *AIProcessedTodo
	aiProcessed := false

	// First, try to use user's configured AI provider
	if s.aiProviderService != nil {
		provider, err := s.aiProviderService.GetDefaultByUserID(userID)
		if err == nil && provider != nil && provider.SelectedModel != nil {
			// Get decrypted API key
			apiKey, err := s.aiProviderService.GetDecryptedAPIKey(provider)
			if err == nil {
				config := &AIProviderConfig{
					ProviderType: provider.ProviderType,
					BaseURL:      provider.BaseURL,
					APIKey:       apiKey,
					Model:        *provider.SelectedModel,
				}
				result, err := ProcessTodoWithProvider(req.Title, config)
				if err == nil && result != nil {
					aiResult = result
					aiProcessed = true
				}
			}
		}
	}

	// Fall back to default AI service from env if user provider didn't work
	if !aiProcessed && s.aiService != nil && s.aiService.IsConfigured() {
		result, err := s.aiService.ProcessTodo(req.Title)
		if err == nil && result != nil {
			aiResult = result
			aiProcessed = true
		}
	}

	// Use AI results or fall back to original input
	// Note: Date is parsed by frontend, AI only handles title cleanup and tags
	title := req.Title
	tags := []string{}
	dueDate := req.DueDate // Frontend is source of truth for dates

	if aiProcessed && aiResult != nil {
		title = aiResult.Title
		tags = aiResult.Tags
	}

	if dueDate != nil {
		log.Printf("[TodoService] Creating todo - title: %q, tags: %v, dueDate: %q", title, tags, *dueDate)
	} else {
		log.Printf("[TodoService] Creating todo - title: %q, tags: %v, dueDate: nil", title, tags)
	}

	todo := &models.Todo{
		UserID:      userID,
		GroupID:     req.GroupID,
		Title:       title,
		Description: req.Description,
		DueDate:     dueDate,
		Priority:    req.Priority,
		Position:    fmt.Sprintf("%d", maxPos+1000),
		Tags:        tags,
	}

	if err := s.todoRepo.Create(todo); err != nil {
		return nil, err
	}

	// Async RAG indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		go func(t *models.Todo) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexTodo(ctx, t); err != nil {
				log.Printf("[TodoService] Failed to index todo %s: %v", t.ID, err)
			}
		}(todo)
	}

	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserTodos(userID)
	}

	return todo, nil
}

// CreateFromChat creates a todo from chat interface (parses content to extract title/description/date)
func (s *TodoService) CreateFromChat(userID string, req *models.TodoCreateFromChatRequest) (*models.Todo, error) {
	log.Printf("[TodoService] Creating todo from chat for user %s: %q", userID, req.Content)

	// Get max position for ordering
	maxPos, err := s.todoRepo.GetMaxPosition(userID)
	if err != nil {
		return nil, err
	}

	// Determine title and description
	var title string
	var description *string
	var dueDate *string = req.DueDate // Use provided due date if available

	// If explicit title provided, use it
	if req.Title != nil && *req.Title != "" {
		title = *req.Title
		if req.Description != nil {
			description = req.Description
		} else {
			// Use content as description if title is explicit
			if req.Content != "" {
				desc := req.Content
				description = &desc
			}
		}
	} else {
		// Parse content to extract title and description
		// Try to extract date from content if not provided
		if dueDate == nil {
			// Simple date extraction - look for common patterns
			// This is basic - could be enhanced with AI or better parsing
			extractedDate := extractDateFromText(req.Content)
			if extractedDate != nil {
				dueDate = extractedDate
			}
		}

		// Use first line or first 50 chars as title
		lines := strings.Split(req.Content, "\n")
		if len(lines) > 0 && lines[0] != "" {
			title = strings.TrimSpace(lines[0])
			if len(title) > 200 {
				title = title[:200]
			}
		} else {
			title = req.Content
			if len(title) > 200 {
				title = title[:200]
			}
		}

		// Use remaining lines as description
		if len(lines) > 1 {
			desc := strings.TrimSpace(strings.Join(lines[1:], "\n"))
			if desc != "" {
				description = &desc
			}
		}
	}

	// Process with AI if available (for title cleanup and tags)
	var aiResult *AIProcessedTodo
	aiProcessed := false

	// First, try to use user's configured AI provider
	if s.aiProviderService != nil {
		provider, err := s.aiProviderService.GetDefaultByUserID(userID)
		if err == nil && provider != nil && provider.SelectedModel != nil {
			apiKey, err := s.aiProviderService.GetDecryptedAPIKey(provider)
			if err == nil {
				config := &AIProviderConfig{
					ProviderType: provider.ProviderType,
					BaseURL:      provider.BaseURL,
					APIKey:       apiKey,
					Model:        *provider.SelectedModel,
				}
				result, err := ProcessTodoWithProvider(title, config)
				if err == nil && result != nil {
					aiResult = result
					aiProcessed = true
				}
			}
		}
	}

	// Fall back to default AI service from env if user provider didn't work
	if !aiProcessed && s.aiService != nil && s.aiService.IsConfigured() {
		result, err := s.aiService.ProcessTodo(title)
		if err == nil && result != nil {
			aiResult = result
			aiProcessed = true
		}
	}

	// Use AI results if available
	tags := []string{}
	if aiProcessed && aiResult != nil {
		title = aiResult.Title
		tags = aiResult.Tags
	}

	// Use provided priority or default to medium
	priority := req.Priority
	if priority == "" {
		priority = models.PriorityMedium
	}

	todo := &models.Todo{
		UserID:      userID,
		GroupID:     req.GroupID,
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		Priority:    priority,
		Position:    fmt.Sprintf("%d", maxPos+1000),
		Tags:        tags,
	}

	if err := s.todoRepo.Create(todo); err != nil {
		return nil, err
	}

	// Async RAG indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		go func(t *models.Todo) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexTodo(ctx, t); err != nil {
				log.Printf("[TodoService] Failed to index todo %s: %v", t.ID, err)
			}
		}(todo)
	}

	log.Printf("[TodoService] Created todo from chat %s: %q", todo.ID, todo.Title)

	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserTodos(userID)
	}

	return todo, nil
}

// extractDateFromText attempts to extract a date from text in various formats
// Returns date in SQL datetime format: "2006-01-02T15:04:05Z" or "2006-01-02 15:04:05"
func extractDateFromText(text string) *string {
	if text == "" {
		return nil
	}

	textLower := strings.ToLower(strings.TrimSpace(text))
	now := time.Now()
	var targetDate time.Time

	// Handle natural language relative dates
	switch {
	case strings.Contains(textLower, "today"):
		targetDate = now
	case strings.Contains(textLower, "tomorrow"):
		targetDate = now.AddDate(0, 0, 1)
	case strings.Contains(textLower, "yesterday"):
		targetDate = now.AddDate(0, 0, -1)
	case strings.Contains(textLower, "next week"):
		targetDate = now.AddDate(0, 0, 7)
	case strings.Contains(textLower, "next month"):
		targetDate = now.AddDate(0, 1, 0)
	case strings.Contains(textLower, "next year"):
		targetDate = now.AddDate(1, 0, 0)
	case strings.Contains(textLower, "in ") && strings.Contains(textLower, " day"):
		// Extract number of days: "in 3 days", "in 5 days"
		re := regexp.MustCompile(`in\s+(\d+)\s+days?`)
		matches := re.FindStringSubmatch(textLower)
		if len(matches) > 1 {
			var days int
			fmt.Sscanf(matches[1], "%d", &days)
			targetDate = now.AddDate(0, 0, days)
		}
	case strings.Contains(textLower, "in ") && strings.Contains(textLower, " week"):
		// Extract number of weeks: "in 2 weeks", "in 3 weeks"
		re := regexp.MustCompile(`in\s+(\d+)\s+weeks?`)
		matches := re.FindStringSubmatch(textLower)
		if len(matches) > 1 {
			var weeks int
			fmt.Sscanf(matches[1], "%d", &weeks)
			targetDate = now.AddDate(0, 0, weeks*7)
		}
	case strings.Contains(textLower, "in ") && strings.Contains(textLower, " month"):
		// Extract number of months: "in 2 months", "in 3 months"
		re := regexp.MustCompile(`in\s+(\d+)\s+months?`)
		matches := re.FindStringSubmatch(textLower)
		if len(matches) > 1 {
			var months int
			fmt.Sscanf(matches[1], "%d", &months)
			targetDate = now.AddDate(0, months, 0)
		}
	case strings.Contains(textLower, "next monday") || strings.Contains(textLower, "next mon"):
		targetDate = nextWeekday(now, time.Monday)
	case strings.Contains(textLower, "next tuesday") || strings.Contains(textLower, "next tue"):
		targetDate = nextWeekday(now, time.Tuesday)
	case strings.Contains(textLower, "next wednesday") || strings.Contains(textLower, "next wed"):
		targetDate = nextWeekday(now, time.Wednesday)
	case strings.Contains(textLower, "next thursday") || strings.Contains(textLower, "next thu"):
		targetDate = nextWeekday(now, time.Thursday)
	case strings.Contains(textLower, "next friday") || strings.Contains(textLower, "next fri"):
		targetDate = nextWeekday(now, time.Friday)
	case strings.Contains(textLower, "next saturday") || strings.Contains(textLower, "next sat"):
		targetDate = nextWeekday(now, time.Saturday)
	case strings.Contains(textLower, "next sunday") || strings.Contains(textLower, "next sun"):
		targetDate = nextWeekday(now, time.Sunday)
	default:
		// Try to parse common date formats
		dateFormats := []string{
			"2006-01-02",
			"2006/01/02",
			"01/02/2006",
			"02/01/2006", // DD/MM/YYYY
			"01-02-2006",
			"02-01-2006", // DD-MM-YYYY
			"January 2, 2006",
			"Jan 2, 2006",
			"2 January 2006",
			"2 Jan 2006",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}

		// Look for date patterns in the text
		datePatterns := []*regexp.Regexp{
			regexp.MustCompile(`\d{4}-\d{2}-\d{2}`),                    // YYYY-MM-DD
			regexp.MustCompile(`\d{2}/\d{2}/\d{4}`),                   // MM/DD/YYYY or DD/MM/YYYY
			regexp.MustCompile(`\d{2}-\d{2}-\d{4}`),                   // MM-DD-YYYY or DD-MM-YYYY
			regexp.MustCompile(`(?i)(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}`), // Jan 15, 2024 (case-insensitive)
			regexp.MustCompile(`(?i)\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4}`), // 15 Jan 2024 (case-insensitive)
		}

		for _, pattern := range datePatterns {
			matches := pattern.FindStringSubmatch(text)
			if len(matches) > 0 {
				dateStr := matches[0]
				for _, format := range dateFormats {
					parsed, err := time.Parse(format, dateStr)
					if err == nil {
						targetDate = parsed
						// If no time specified, set to end of day (23:59:59)
						if !strings.Contains(format, "15:04:05") {
							targetDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 0, targetDate.Location())
						}
						break
					}
				}
				if !targetDate.IsZero() {
					break
				}
			}
		}
	}

	// If we found a date, format it as SQL datetime
	if !targetDate.IsZero() {
		// Format as SQL datetime: "2006-01-02T15:04:05Z"
		formatted := targetDate.Format("2006-01-02T15:04:05Z")
		return &formatted
	}

	return nil
}

// nextWeekday returns the next occurrence of the specified weekday
func nextWeekday(from time.Time, weekday time.Weekday) time.Time {
	daysUntil := int(weekday - from.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7 // Next week
	}
	return from.AddDate(0, 0, daysUntil)
}

func (s *TodoService) GetAll(userID string) ([]models.Todo, error) {
	// Try to get from cache first
	if s.cacheService != nil {
		cached, err := s.cacheService.GetCachedUserTodos(userID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Cache miss or Redis unavailable - fetch from database
	todos, err := s.todoRepo.GetAllByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result (async, don't block)
	if s.cacheService != nil {
		go func() {
			_ = s.cacheService.CacheUserTodos(userID, todos)
		}()
	}

	return todos, nil
}

func (s *TodoService) GetByID(userID, todoID string) (*models.Todo, error) {
	todo, err := s.todoRepo.GetByID(todoID)
	if err != nil {
		return nil, err
	}
	if todo == nil || todo.UserID != userID {
		return nil, nil
	}
	return todo, nil
}

func (s *TodoService) Update(userID, todoID string, req *models.TodoUpdateRequest) (*models.Todo, error) {
	// Verify ownership
	todo, err := s.todoRepo.GetByID(todoID)
	if err != nil {
		return nil, err
	}
	if todo == nil || todo.UserID != userID {
		return nil, fmt.Errorf("todo not found")
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.GroupID != nil {
		updates["group_id"] = *req.GroupID
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}

	if len(updates) > 0 {
		if err := s.todoRepo.Update(todoID, updates); err != nil {
			return nil, err
		}
	}

	updatedTodo, err := s.todoRepo.GetByID(todoID)
	if err != nil {
		return nil, err
	}

	// Async RAG indexing - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() && updatedTodo != nil {
		go func(t *models.Todo) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.IndexTodo(ctx, t); err != nil {
				log.Printf("[TodoService] Failed to re-index todo %s: %v", t.ID, err)
			}
		}(updatedTodo)
	}

	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserTodos(userID)
	}

	return updatedTodo, nil
}

func (s *TodoService) Delete(userID, todoID string) error {
	// Verify ownership
	todo, err := s.todoRepo.GetByID(todoID)
	if err != nil {
		return err
	}
	if todo == nil || todo.UserID != userID {
		return fmt.Errorf("todo not found")
	}

	// Async RAG deletion - fire and forget
	if s.ragService != nil && s.ragService.IsConfigured() {
		go func(id string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.ragService.DeleteFromIndex(ctx, models.ContentTypeTodo, id); err != nil {
				log.Printf("[TodoService] Failed to delete todo %s from index: %v", id, err)
			}
		}(todoID)
	}

	if err := s.todoRepo.Delete(todoID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cacheService != nil {
		_ = s.cacheService.InvalidateUserTodos(userID)
	}

	return nil
}

func (s *TodoService) Reorder(userID string, req *models.TodoReorderRequest) error {
	// Verify all todos belong to user before updating
	for _, t := range req.Todos {
		todo, err := s.todoRepo.GetByID(t.ID)
		if err != nil {
			return err
		}
		if todo == nil || todo.UserID != userID {
			return fmt.Errorf("todo %s not found or unauthorized", t.ID)
		}
	}

	return s.todoRepo.UpdatePositions(req.Todos)
}
