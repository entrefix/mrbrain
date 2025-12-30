package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/services"
)

type GroupHandler struct {
	groupService *services.GroupService
}

func NewGroupHandler(groupService *services.GroupService) *GroupHandler {
	return &GroupHandler{
		groupService: groupService,
	}
}

func (h *GroupHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)

	groups, err := h.groupService.GetAll(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
	})
}

func (h *GroupHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.groupService.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"group": group,
	})
}

func (h *GroupHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupID := c.Param("id")

	group, err := h.groupService.GetByID(userID, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch group"})
		return
	}
	if group == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"group": group,
	})
}

func (h *GroupHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupID := c.Param("id")

	var req models.GroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.groupService.Update(userID, groupID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"group": group,
	})
}

func (h *GroupHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupID := c.Param("id")

	if err := h.groupService.Delete(userID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "group deleted successfully",
	})
}
