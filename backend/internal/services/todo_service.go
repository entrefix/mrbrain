package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

type TodoService struct {
	todoRepo          *repository.TodoRepository
	aiService         *AIService
	aiProviderService *AIProviderService
	ragService        *RAGService
}

func NewTodoService(todoRepo *repository.TodoRepository, aiService *AIService, aiProviderService *AIProviderService, ragService *RAGService) *TodoService {
	return &TodoService{
		todoRepo:          todoRepo,
		aiService:         aiService,
		aiProviderService: aiProviderService,
		ragService:        ragService,
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

	return todo, nil
}

func (s *TodoService) GetAll(userID string) ([]models.Todo, error) {
	return s.todoRepo.GetAllByUserID(userID)
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

	return s.todoRepo.Delete(todoID)
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
