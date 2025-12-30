package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/services"
)

type UserDataHandler struct {
	userDataService *services.UserDataService
}

func NewUserDataHandler(userDataService *services.UserDataService) *UserDataHandler {
	return &UserDataHandler{
		userDataService: userDataService,
	}
}

// GetDataStats returns counts of user data
func (h *UserDataHandler) GetDataStats(c *gin.Context) {
	userID := middleware.GetUserID(c)

	stats, err := h.userDataService.GetDataStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get data stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ClearMemories deletes all memories for the user
func (h *UserDataHandler) ClearMemories(c *gin.Context) {
	userID := middleware.GetUserID(c)

	result, err := h.userDataService.ClearAllMemories(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to clear memories",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ClearAllData deletes all todos, memories, and custom groups for the user
func (h *UserDataHandler) ClearAllData(c *gin.Context) {
	userID := middleware.GetUserID(c)

	result, err := h.userDataService.ClearAllData(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to clear all data",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
