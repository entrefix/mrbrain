package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/services"
)

type ChatHandler struct {
	chatService *services.ChatService
}

func NewChatHandler(chatService *services.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// GetActiveThread returns or creates the active thread for the user
func (h *ChatHandler) GetActiveThread(c *gin.Context) {
	userID := middleware.GetUserID(c)

	thread, err := h.chatService.GetOrCreateActiveThread(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active thread"})
		return
	}

	// Get messages for this thread
	messages, err := h.chatService.GetThreadWithMessages(userID, thread.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// GetAllThreads returns all threads for the user
func (h *ChatHandler) GetAllThreads(c *gin.Context) {
	userID := middleware.GetUserID(c)

	threads, err := h.chatService.GetAllThreads(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch threads"})
		return
	}

	c.JSON(http.StatusOK, models.ChatThreadsResponse{
		Threads: threads,
	})
}

// GetThread returns a thread with all its messages
func (h *ChatHandler) GetThread(c *gin.Context) {
	userID := middleware.GetUserID(c)
	threadID := c.Param("id")

	response, err := h.chatService.GetThreadWithMessages(userID, threadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "thread not found"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateThread creates a new thread
func (h *ChatHandler) CreateThread(c *gin.Context) {
	userID := middleware.GetUserID(c)

	thread, err := h.chatService.CreateThread(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create thread"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"thread": thread,
	})
}

// AddMessage adds a message to a thread
func (h *ChatHandler) AddMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	threadID := c.Param("id")

	var req models.ChatMessageCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.chatService.AddMessage(userID, threadID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": message,
	})
}

// DeleteThread deletes a thread
func (h *ChatHandler) DeleteThread(c *gin.Context) {
	userID := middleware.GetUserID(c)
	threadID := c.Param("id")

	if err := h.chatService.DeleteThread(userID, threadID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "thread deleted successfully",
	})
}

