package models

import "time"

type Priority string
type Status string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
)

type Todo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	GroupID     *string   `json:"group_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	DueDate     *string   `json:"due_date"`
	Priority    Priority  `json:"priority"`
	Status      Status    `json:"status"`
	Position    string    `json:"position"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TodoCreateRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description *string  `json:"description"`
	DueDate     *string  `json:"due_date"`
	Priority    Priority `json:"priority"`
	GroupID     *string  `json:"group_id"`
}

type TodoCreateFromChatRequest struct {
	Content     string   `json:"content" binding:"required"` // Content to parse for title/description
	Title       *string  `json:"title"`                      // Optional explicit title
	Description *string  `json:"description"`                // Optional explicit description
	DueDate     *string  `json:"due_date"`                   // Optional due date
	Priority    Priority `json:"priority"`                    // Optional priority
	GroupID     *string  `json:"group_id"`                   // Optional group
}

type TodoUpdateRequest struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	DueDate     *string   `json:"due_date"`
	Priority    *Priority `json:"priority"`
	Status      *Status   `json:"status"`
	GroupID     *string   `json:"group_id"`
	Position    *string   `json:"position"`
	Tags        []string  `json:"tags"`
}

type TodoReorderRequest struct {
	Todos []TodoPosition `json:"todos" binding:"required"`
}

type TodoPosition struct {
	ID       string `json:"id" binding:"required"`
	Position string `json:"position" binding:"required"`
}
