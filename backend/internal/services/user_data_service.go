package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/todomyday/backend/internal/repository"
)

type UserDataService struct {
	memoryRepo *repository.MemoryRepository
	todoRepo   *repository.TodoRepository
	groupRepo  *repository.GroupRepository
	ragService *RAGService
}

func NewUserDataService(
	memoryRepo *repository.MemoryRepository,
	todoRepo *repository.TodoRepository,
	groupRepo *repository.GroupRepository,
	ragService *RAGService,
) *UserDataService {
	return &UserDataService{
		memoryRepo: memoryRepo,
		todoRepo:   todoRepo,
		groupRepo:  groupRepo,
		ragService: ragService,
	}
}

// ClearMemoriesResult holds the result of clearing memories
type ClearMemoriesResult struct {
	MemoriesDeleted int    `json:"memories_deleted"`
	Success         bool   `json:"success"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// ClearAllResult holds the result of clearing all data
type ClearAllResult struct {
	MemoriesDeleted     int    `json:"memories_deleted"`
	TodosDeleted        int    `json:"todos_deleted"`
	CustomGroupsDeleted int    `json:"custom_groups_deleted"`
	Success             bool   `json:"success"`
	ErrorMessage        string `json:"error_message,omitempty"`
}

// DataStats holds counts of user data for confirmation UI
type DataStats struct {
	MemoryCount      int `json:"memory_count"`
	TodoCount        int `json:"todo_count"`
	CustomGroupCount int `json:"custom_group_count"`
}

// ClearAllMemories deletes all memories for a user
func (s *UserDataService) ClearAllMemories(userID string) (*ClearMemoriesResult, error) {
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("[UserDataService] Starting ClearAllMemories for user: %s", userID)

	result := &ClearMemoriesResult{Success: false}

	// Note: ClaraVector documents are not deleted here since we don't track document IDs
	// The next reindex will overwrite old data

	// Delete from SQL DB (authoritative)
	count, err := s.memoryRepo.CountByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count memories: %w", err)
	}

	rowsAffected, err := s.memoryRepo.DeleteAllByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete memories: %w", err)
	}

	result.MemoriesDeleted = int(rowsAffected)
	result.Success = true

	log.Printf("[UserDataService] ClearAllMemories complete: deleted %d memories (expected: %d)", rowsAffected, count)

	return result, nil
}

// ClearAllData deletes all todos, memories, and custom groups for a user
// Keeps: AI providers, default groups
// Deletes: Custom groups, all todos, all memories
func (s *UserDataService) ClearAllData(userID string) (*ClearAllResult, error) {
	_, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("[UserDataService] Starting ClearAllData for user: %s", userID)

	result := &ClearAllResult{Success: false}

	// Note: ClaraVector documents are not deleted here since we don't track document IDs
	// The user can trigger a reindex to rebuild the vector index

	// Delete memories from SQL
	memoriesDeleted, err := s.memoryRepo.DeleteAllByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete memories: %w", err)
	}
	result.MemoriesDeleted = int(memoriesDeleted)
	log.Printf("[UserDataService] Deleted %d memories", memoriesDeleted)

	// Delete todos from SQL
	todosDeleted, err := s.todoRepo.DeleteAllByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete todos: %w", err)
	}
	result.TodosDeleted = int(todosDeleted)
	log.Printf("[UserDataService] Deleted %d todos", todosDeleted)

	// Delete custom groups (NOT default groups)
	groupsDeleted, err := s.groupRepo.DeleteAllCustomByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete custom groups: %w", err)
	}
	result.CustomGroupsDeleted = int(groupsDeleted)
	log.Printf("[UserDataService] Deleted %d custom groups", groupsDeleted)

	result.Success = true
	log.Printf("[UserDataService] ClearAllData complete: memories=%d, todos=%d, groups=%d",
		memoriesDeleted, todosDeleted, groupsDeleted)

	return result, nil
}

// GetDataStats returns counts of user data (for confirmation UI)
func (s *UserDataService) GetDataStats(userID string) (*DataStats, error) {
	stats := &DataStats{}

	memCount, err := s.memoryRepo.CountByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count memories: %w", err)
	}
	stats.MemoryCount = memCount

	todoCount, err := s.todoRepo.CountByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count todos: %w", err)
	}
	stats.TodoCount = todoCount

	groupCount, err := s.groupRepo.CountCustomByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count groups: %w", err)
	}
	stats.CustomGroupCount = groupCount

	return stats, nil
}
