package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/services"
)

type MemoryHandler struct {
	memoryService *services.MemoryService
}

func NewMemoryHandler(memoryService *services.MemoryService) *MemoryHandler {
	return &MemoryHandler{
		memoryService: memoryService,
	}
}

// GetAll returns all memories for the user
func (h *MemoryHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	memories, err := h.memoryService.GetAll(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch memories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
	})
}

// Create creates a new memory
func (h *MemoryHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.MemoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memory, err := h.memoryService.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create memory"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"memory": memory,
	})
}

// GetByID returns a single memory
func (h *MemoryHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	memoryID := c.Param("id")

	memory, err := h.memoryService.GetByID(userID, memoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch memory"})
		return
	}
	if memory == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memory": memory,
	})
}

// Update updates a memory
func (h *MemoryHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	memoryID := c.Param("id")

	var req models.MemoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memory, err := h.memoryService.Update(userID, memoryID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memory": memory,
	})
}

// Delete deletes a memory
func (h *MemoryHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	memoryID := c.Param("id")

	if err := h.memoryService.Delete(userID, memoryID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "memory deleted successfully",
	})
}

// GetCategories returns all available categories
func (h *MemoryHandler) GetCategories(c *gin.Context) {
	userID := middleware.GetUserID(c)

	categories, err := h.memoryService.GetCategories(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
	})
}

// GetByCategory returns memories filtered by category
func (h *MemoryHandler) GetByCategory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	category := c.Param("category")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	memories, err := h.memoryService.GetByCategory(userID, category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch memories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
	})
}

// Search performs full-text search
func (h *MemoryHandler) Search(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.MemorySearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memories, err := h.memoryService.Search(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search memories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
	})
}

// ConvertToTodo converts a memory to a todo
func (h *MemoryHandler) ConvertToTodo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	memoryID := c.Param("id")

	var req models.MemoryToTodoRequest
	// Binding is optional - can convert without additional params
	c.ShouldBindJSON(&req)

	todo, err := h.memoryService.ConvertToTodo(userID, memoryID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"todo":    todo,
		"message": "memory converted to todo successfully",
	})
}

// GetDigest returns the weekly digest
func (h *MemoryHandler) GetDigest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	digest, err := h.memoryService.GetOrGenerateDigest(userID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"digest": digest,
	})
}

// GenerateDigest regenerates the weekly digest
func (h *MemoryHandler) GenerateDigest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	digest, err := h.memoryService.GetOrGenerateDigest(userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"digest": digest,
	})
}

// WebSearch searches the web using SearXNG
func (h *MemoryHandler) WebSearch(c *gin.Context) {
	var req models.WebSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.memoryService.WebSearch(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}

// GetStats returns memory statistics
func (h *MemoryHandler) GetStats(c *gin.Context) {
	userID := middleware.GetUserID(c)

	stats, err := h.memoryService.GetStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
