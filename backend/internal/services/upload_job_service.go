package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/models"
)

// UploadJobService manages async file upload jobs
type UploadJobService struct {
	jobs map[string]*models.UploadJob
	mu   sync.RWMutex
}

// NewUploadJobService creates a new upload job service
func NewUploadJobService() *UploadJobService {
	service := &UploadJobService{
		jobs: make(map[string]*models.UploadJob),
	}
	
	// Start cleanup goroutine to remove old completed jobs (after 1 hour)
	go service.cleanupOldJobs()
	
	return service
}

// CreateJob creates a new upload job
func (s *UploadJobService) CreateJob(userID, filename, fileType string, totalItems int) *models.UploadJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &models.UploadJob{
		ID:             uuid.New().String(),
		UserID:         userID,
		Filename:       filename,
		FileType:       fileType,
		Status:         models.JobStatusPending,
		Progress:       0,
		TotalItems:     totalItems,
		ProcessedItems: 0,
		Memories:       []models.Memory{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	s.jobs[job.ID] = job
	return job
}

// GetJob retrieves a job by ID
func (s *UploadJobService) GetJob(jobID string) (*models.UploadJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	return job, nil
}

// UpdateJobStatus updates the job status
func (s *UploadJobService) UpdateJobStatus(jobID string, status models.UploadJobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	job.Status = status
	job.UpdatedAt = time.Now()

	if status == models.JobStatusCompleted || status == models.JobStatusFailed {
		now := time.Now()
		job.CompletedAt = &now
		job.Progress = 100
	}

	return nil
}

// AddMemoryToJob adds a newly created memory to the job
func (s *UploadJobService) AddMemoryToJob(jobID string, memory models.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	job.Memories = append(job.Memories, memory)
	job.ProcessedItems++
	job.UpdatedAt = time.Now()

	// Update progress
	if job.TotalItems > 0 {
		job.Progress = (job.ProcessedItems * 100) / job.TotalItems
	}

	return nil
}

// SetJobError sets an error message for the job
func (s *UploadJobService) SetJobError(jobID string, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	job.Status = models.JobStatusFailed
	job.ErrorMessage = errorMsg
	job.UpdatedAt = time.Now()
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// cleanupOldJobs removes completed jobs older than 1 hour
func (s *UploadJobService) cleanupOldJobs() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, job := range s.jobs {
			if job.CompletedAt != nil && now.Sub(*job.CompletedAt) > 1*time.Hour {
				delete(s.jobs, id)
			}
		}
		s.mu.Unlock()
	}
}

// GetJobStatus returns the current status of a job
func (s *UploadJobService) GetJobStatus(jobID string) (*models.UploadJobStatusResponse, error) {
	job, err := s.GetJob(jobID)
	if err != nil {
		return nil, err
	}

	return &models.UploadJobStatusResponse{
		JobID:          job.ID,
		Status:         job.Status,
		Progress:       job.Progress,
		TotalItems:     job.TotalItems,
		ProcessedItems: job.ProcessedItems,
		Memories:       job.Memories,
		ErrorMessage:   job.ErrorMessage,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		CompletedAt:    job.CompletedAt,
	}, nil
}






