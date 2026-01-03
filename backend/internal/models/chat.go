package models

import "time"

type ChatThread struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	Role      string    `json:"role"` // 'user' or 'assistant'
	Content   string    `json:"content"`
	Mode      *string   `json:"mode"`      // 'memories', 'internet', 'hybrid', 'llm'
	Sources   *string   `json:"sources"`    // JSON array of sources
	CreatedAt time.Time `json:"created_at"`
}

type ChatThreadCreateRequest struct {
	// Empty for now, can add fields later if needed
}

type ChatMessageCreateRequest struct {
	Role    string  `json:"role" binding:"required"`
	Content string  `json:"content" binding:"required"`
	Mode    *string `json:"mode"`
	Sources *string `json:"sources"`
}

type ChatThreadResponse struct {
	Thread   *ChatThread    `json:"thread"`
	Messages []ChatMessage  `json:"messages"`
}

type ChatThreadsResponse struct {
	Threads []ChatThread `json:"threads"`
}

