package services

import (
	"fmt"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

type ChatService struct {
	chatRepo *repository.ChatRepository
}

func NewChatService(chatRepo *repository.ChatRepository) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
	}
}

// GetOrCreateActiveThread gets the active thread for a user or creates a new one
func (s *ChatService) GetOrCreateActiveThread(userID string) (*models.ChatThread, error) {
	// Try to get the most recent thread
	thread, err := s.chatRepo.GetActiveThreadByUserID(userID)
	if err != nil {
		return nil, err
	}

	// If no thread exists, create a new one
	if thread == nil {
		thread = &models.ChatThread{
			UserID: userID,
		}
		if err := s.chatRepo.CreateThread(thread); err != nil {
			return nil, err
		}
	}

	return thread, nil
}

// GetAllThreads returns all threads for a user
func (s *ChatService) GetAllThreads(userID string) ([]models.ChatThread, error) {
	return s.chatRepo.GetThreadsByUserID(userID)
}

// GetThreadWithMessages returns a thread with all its messages
func (s *ChatService) GetThreadWithMessages(userID, threadID string) (*models.ChatThreadResponse, error) {
	// Verify thread belongs to user
	thread, err := s.chatRepo.GetThreadByID(threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, fmt.Errorf("thread not found")
	}
	if thread.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Get messages
	messages, err := s.chatRepo.GetMessagesByThreadID(threadID)
	if err != nil {
		return nil, err
	}

	return &models.ChatThreadResponse{
		Thread:   thread,
		Messages: messages,
	}, nil
}

// CreateThread creates a new thread for a user
func (s *ChatService) CreateThread(userID string) (*models.ChatThread, error) {
	thread := &models.ChatThread{
		UserID: userID,
	}
	if err := s.chatRepo.CreateThread(thread); err != nil {
		return nil, err
	}
	return thread, nil
}

// AddMessage adds a message to a thread
func (s *ChatService) AddMessage(userID, threadID string, req *models.ChatMessageCreateRequest) (*models.ChatMessage, error) {
	// Verify thread belongs to user
	thread, err := s.chatRepo.GetThreadByID(threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, fmt.Errorf("thread not found")
	}
	if thread.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	message := &models.ChatMessage{
		ThreadID: threadID,
		Role:      req.Role,
		Content:   req.Content,
		Mode:      req.Mode,
		Sources:   req.Sources,
	}

	if err := s.chatRepo.CreateMessage(message); err != nil {
		return nil, err
	}

	return message, nil
}

// DeleteThread deletes a thread (and all its messages via cascade)
func (s *ChatService) DeleteThread(userID, threadID string) error {
	// Verify thread belongs to user
	thread, err := s.chatRepo.GetThreadByID(threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return fmt.Errorf("thread not found")
	}
	if thread.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	return s.chatRepo.DeleteThread(threadID)
}

