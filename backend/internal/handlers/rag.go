package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/services"
)

type RAGHandler struct {
	ragService *services.RAGService
}

func NewRAGHandler(ragService *services.RAGService) *RAGHandler {
	return &RAGHandler{ragService: ragService}
}

// Search performs hybrid search across todos and memories
// POST /api/rag/search
func (h *RAGHandler) Search(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if h.ragService == nil || !h.ragService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "RAG service not configured",
			"message": "Please configure embedding API settings",
		})
		return
	}

	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	resp, err := h.ragService.Search(c.Request.Context(), userID, &req)
	if err != nil {
		log.Printf("[RAG Handler] Search error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Ask answers a question using RAG
// POST /api/rag/ask
func (h *RAGHandler) Ask(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if h.ragService == nil || !h.ragService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "RAG service not configured",
			"message": "Please configure embedding API settings",
		})
		return
	}

	var req models.AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
		return
	}

	resp, err := h.ragService.Ask(c.Request.Context(), userID, &req)
	if err != nil {
		log.Printf("[RAG Handler] Ask error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to answer question"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// IndexAll indexes all content for the current user
// POST /api/rag/index
func (h *RAGHandler) IndexAll(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if h.ragService == nil || !h.ragService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "RAG service not configured",
			"message": "Please configure embedding API settings",
		})
		return
	}

	resp, err := h.ragService.IndexAllForUser(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[RAG Handler] Index error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "indexing failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetStats returns index statistics
// GET /api/rag/stats
func (h *RAGHandler) GetStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if h.ragService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	stats := h.ragService.GetStats(userID)

	c.JSON(http.StatusOK, gin.H{
		"configured": h.ragService.IsConfigured(),
		"stats":      stats,
	})
}
