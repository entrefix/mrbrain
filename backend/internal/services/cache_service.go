package services

import (
	"fmt"
	"time"

	"github.com/todomyday/backend/internal/models"
)

type CacheService struct {
	redisService *RedisService
	ttlTodos     time.Duration
	ttlMemories  time.Duration
	ttlAI        time.Duration
}

func NewCacheService(redisService *RedisService, ttlTodos, ttlMemories, ttlAI time.Duration) *CacheService {
	return &CacheService{
		redisService: redisService,
		ttlTodos:     ttlTodos,
		ttlMemories:  ttlMemories,
		ttlAI:        ttlAI,
	}
}

// User Todos Caching

func (s *CacheService) CacheUserTodos(userID string, todos []models.Todo) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:user:%s:todos", userID)
	return s.redisService.SetJSON(key, todos, s.ttlTodos)
}

func (s *CacheService) GetCachedUserTodos(userID string) ([]models.Todo, error) {
	if !s.redisService.IsEnabled() {
		return nil, nil // Graceful degradation - return nil to indicate cache miss
	}

	key := fmt.Sprintf("cache:user:%s:todos", userID)
	var todos []models.Todo
	err := s.redisService.GetJSON(key, &todos)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}
	return todos, nil
}

func (s *CacheService) InvalidateUserTodos(userID string) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:user:%s:todos", userID)
	return s.redisService.Delete(key)
}

// User Memories Caching

func (s *CacheService) CacheUserMemories(userID string, memories []models.Memory) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:user:%s:memories", userID)
	return s.redisService.SetJSON(key, memories, s.ttlMemories)
}

func (s *CacheService) GetCachedUserMemories(userID string) ([]models.Memory, error) {
	if !s.redisService.IsEnabled() {
		return nil, nil // Graceful degradation - return nil to indicate cache miss
	}

	key := fmt.Sprintf("cache:user:%s:memories", userID)
	var memories []models.Memory
	err := s.redisService.GetJSON(key, &memories)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}
	return memories, nil
}

func (s *CacheService) InvalidateUserMemories(userID string) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:user:%s:memories", userID)
	return s.redisService.Delete(key)
}

// AI Response Caching

func (s *CacheService) CacheAIResponse(cacheKey string, response interface{}) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:ai:%s", cacheKey)
	return s.redisService.SetJSON(key, response, s.ttlAI)
}

func (s *CacheService) GetCachedAIResponse(cacheKey string, dest interface{}) error {
	if !s.redisService.IsEnabled() {
		return fmt.Errorf("cache miss") // Return error to indicate cache miss
	}

	key := fmt.Sprintf("cache:ai:%s", cacheKey)
	return s.redisService.GetJSON(key, dest)
}

func (s *CacheService) InvalidateAIResponse(cacheKey string) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:ai:%s", cacheKey)
	return s.redisService.Delete(key)
}

// RAG Search Result Caching

func (s *CacheService) CacheRAGSearch(cacheKey string, results interface{}) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	key := fmt.Sprintf("cache:rag:search:%s", cacheKey)
	return s.redisService.SetJSON(key, results, s.ttlAI)
}

func (s *CacheService) GetCachedRAGSearch(cacheKey string, dest interface{}) error {
	if !s.redisService.IsEnabled() {
		return fmt.Errorf("cache miss") // Return error to indicate cache miss
	}

	key := fmt.Sprintf("cache:rag:search:%s", cacheKey)
	return s.redisService.GetJSON(key, dest)
}

// Invalidate all user cache

func (s *CacheService) InvalidateUserCache(userID string) error {
	if !s.redisService.IsEnabled() {
		return nil // Graceful degradation
	}

	// Invalidate todos
	if err := s.InvalidateUserTodos(userID); err != nil {
		return err
	}

	// Invalidate memories
	if err := s.InvalidateUserMemories(userID); err != nil {
		return err
	}

	return nil
}
