package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/services"
)

type MemoryHandler struct {
	memoryService     *services.MemoryService
	fileParserService *services.FileParserService
	uploadJobService  *services.UploadJobService
	visionService     *services.VisionService
}

func NewMemoryHandler(memoryService *services.MemoryService, fileParserService *services.FileParserService, uploadJobService *services.UploadJobService, visionService *services.VisionService) *MemoryHandler {
	return &MemoryHandler{
		memoryService:     memoryService,
		fileParserService: fileParserService,
		uploadJobService:  uploadJobService,
		visionService:     visionService,
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

// Reorder updates positions for multiple memories
func (h *MemoryHandler) Reorder(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.MemoryReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.memoryService.Reorder(userID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "memories reordered successfully",
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

// UploadMemoryFile handles file upload for creating memories asynchronously
// Returns a job ID that can be polled for progress
func (h *MemoryHandler) UploadMemoryFile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// 1. Get uploaded file from multipart form-data
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// 2. Validate file (type and size)
	if err := h.fileParserService.ValidateFile(file.Filename, file.Size); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Open and read file content
	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer fileContent.Close()

	contentBytes, err := io.ReadAll(fileContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
		return
	}

	// 4. Parse file into memory sections
	sections, err := h.fileParserService.ParseFile(file.Filename, contentBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Parse error: %v", err)})
		return
	}

	// 5. Create a job for async processing
	fileType := filepath.Ext(file.Filename)
	job := h.uploadJobService.CreateJob(userID, file.Filename, fileType, len(sections))

	log.Printf("[UploadMemoryFile] Created job %s for user %s with %d sections", job.ID, userID, len(sections))

	// 6. Process sections asynchronously
	go h.processUploadJob(job.ID, userID, sections)

	// 7. Return job ID immediately
	c.JSON(http.StatusAccepted, models.UploadJobCreateResponse{
		JobID:    job.ID,
		Status:   job.Status,
		Filename: job.Filename,
		FileType: job.FileType,
	})
}

// processUploadJob processes file sections asynchronously and updates job status
func (h *MemoryHandler) processUploadJob(jobID, userID string, sections []services.ParsedMemorySection) {
	// Update job status to processing
	h.uploadJobService.UpdateJobStatus(jobID, models.JobStatusProcessing)

	log.Printf("[UploadJob:%s] Starting processing of %d sections", jobID, len(sections))

	for i, section := range sections {
		req := &models.MemoryCreateRequest{
			Content: section.Content,
		}

		memory, err := h.memoryService.Create(userID, req)
		if err != nil {
			log.Printf("[UploadJob:%s] Failed to create memory for section %d %q: %v", jobID, i+1, section.Heading, err)
			// Continue with other sections even if one fails
			continue
		}

		// Add memory to job (this updates the memories list progressively)
		if err := h.uploadJobService.AddMemoryToJob(jobID, *memory); err != nil {
			log.Printf("[UploadJob:%s] Failed to add memory to job: %v", jobID, err)
		}

		log.Printf("[UploadJob:%s] Processed section %d/%d: %q", jobID, i+1, len(sections), section.Heading)
	}

	// Mark job as completed
	h.uploadJobService.UpdateJobStatus(jobID, models.JobStatusCompleted)
	log.Printf("[UploadJob:%s] Completed processing", jobID)
}

// GetUploadJobStatus returns the current status of an upload job
func (h *MemoryHandler) GetUploadJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	status, err := h.uploadJobService.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// UploadImage handles image upload and extracts notes/details using GLM-4.5V
func (h *MemoryHandler) UploadImage(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// Check if vision service is configured
	if h.visionService == nil || !h.visionService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Vision service not configured"})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image uploaded"})
		return
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	// Also check by extension if content-type is not reliable
	ext := filepath.Ext(file.Filename)
	validExts := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
	}

	if !validTypes[contentType] {
		if mimeType, ok := validExts[ext]; ok {
			contentType = mimeType
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type. Supported: JPG, PNG, GIF, WebP"})
			return
		}
	}

	// Validate file size (max 10MB)
	maxSize := int64(10 * 1024 * 1024)
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image too large. Maximum size is 10MB"})
		return
	}

	// Read file content
	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
		return
	}
	defer fileContent.Close()

	imageData, err := io.ReadAll(fileContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image content"})
		return
	}

	log.Printf("[UploadImage] Processing image for user %s: %s (%s, %d bytes)", userID, file.Filename, contentType, len(imageData))

	// Process image with vision service
	visionResult, err := h.visionService.ProcessImage(imageData, contentType)
	if err != nil {
		log.Printf("[UploadImage] Vision processing failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process image: %v", err)})
		return
	}

	// Create memory from extracted content
	req := &models.MemoryCreateRequest{
		Content: visionResult.Content,
	}

	memory, err := h.memoryService.CreateWithCategory(userID, req, visionResult.Category, visionResult.Summary)
	if err != nil {
		log.Printf("[UploadImage] Failed to create memory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save memory"})
		return
	}

	log.Printf("[UploadImage] Created memory %s from image with category %s", memory.ID, memory.Category)

	c.JSON(http.StatusCreated, gin.H{
		"memory":        memory,
		"vision_result": visionResult,
	})
}
