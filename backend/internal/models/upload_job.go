package models

import "time"

// UploadJobStatus represents the status of a file upload job
type UploadJobStatus string

const (
	JobStatusPending    UploadJobStatus = "pending"
	JobStatusProcessing UploadJobStatus = "processing"
	JobStatusCompleted  UploadJobStatus = "completed"
	JobStatusFailed     UploadJobStatus = "failed"
)

// UploadJob represents an asynchronous file upload job
type UploadJob struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Filename       string          `json:"filename"`
	FileType       string          `json:"file_type"`
	Status         UploadJobStatus `json:"status"`
	Progress       int             `json:"progress"`        // 0-100
	TotalItems     int             `json:"total_items"`     // Total number of items to process
	ProcessedItems int             `json:"processed_items"` // Number of items processed so far
	Memories       []Memory        `json:"memories"`        // List of created memories (updated progressively)
	ErrorMessage   string          `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// UploadJobCreateResponse is returned when a new upload job is created
type UploadJobCreateResponse struct {
	JobID    string          `json:"job_id"`
	Status   UploadJobStatus `json:"status"`
	Filename string          `json:"filename"`
	FileType string          `json:"file_type"`
}

// UploadJobStatusResponse is returned when checking job status
type UploadJobStatusResponse struct {
	JobID          string          `json:"job_id"`
	Status         UploadJobStatus `json:"status"`
	Progress       int             `json:"progress"`
	TotalItems     int             `json:"total_items"`
	ProcessedItems int             `json:"processed_items"`
	Memories       []Memory        `json:"memories"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}






