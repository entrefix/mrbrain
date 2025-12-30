package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/services"
)

type AIProviderHandler struct {
	service *services.AIProviderService
}

func NewAIProviderHandler(service *services.AIProviderService) *AIProviderHandler {
	return &AIProviderHandler{service: service}
}

func (h *AIProviderHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input models.AIProviderCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.service.Create(userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

func (h *AIProviderHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providers, err := h.service.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if providers == nil {
		providers = []models.AIProvider{}
	}

	c.JSON(http.StatusOK, providers)
}

func (h *AIProviderHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id := c.Param("id")
	provider, err := h.service.GetByID(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func (h *AIProviderHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id := c.Param("id")
	var input models.AIProviderUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.service.Update(id, userID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func (h *AIProviderHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id := c.Param("id")
	if err := h.service.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

func (h *AIProviderHandler) TestConnection(c *gin.Context) {
	var input models.TestConnectionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.TestConnection(&input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AIProviderHandler) FetchModels(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id := c.Param("id")
	fetchedModels, err := h.service.FetchAndSaveModels(id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fetchedModels)
}

func (h *AIProviderHandler) GetModels(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id := c.Param("id")
	providerModels, err := h.service.GetModels(id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if providerModels == nil {
		providerModels = []models.AIProviderModel{}
	}

	c.JSON(http.StatusOK, providerModels)
}
