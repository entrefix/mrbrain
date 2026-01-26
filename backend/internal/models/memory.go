package models

import "time"

type Memory struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Content    string    `json:"content"`
	Summary    *string   `json:"summary"`
	Category   string    `json:"category"`
	URL        *string   `json:"url"`
	URLTitle   *string   `json:"url_title"`
	URLContent *string   `json:"url_content"`
	IsArchived bool      `json:"is_archived"`
	Position   string    `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type MemoryCategory struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"user_id"`
	Name      string    `json:"name"`
	ColorCode string    `json:"color_code"`
	Icon      *string   `json:"icon"`
	IsSystem  bool      `json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
}

type MemoryDigest struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	WeekStart     string    `json:"week_start"`
	WeekEnd       string    `json:"week_end"`
	DigestContent string    `json:"digest_content"`
	CreatedAt     time.Time `json:"created_at"`
}

type MemoryCreateRequest struct {
	Content string `json:"content" binding:"required"`
}

type MemoryCreateFromChatRequest struct {
	Content  string  `json:"content" binding:"required"`
	Category *string `json:"category"`
	Summary  *string `json:"summary"`
}

type MemoryUpdateRequest struct {
	Content    *string `json:"content"`
	Category   *string `json:"category"`
	IsArchived *bool   `json:"is_archived"`
}

type MemorySearchRequest struct {
	Query    string  `json:"query"`
	Category *string `json:"category"`
	DateFrom *string `json:"date_from"`
	DateTo   *string `json:"date_to"`
	Limit    int     `json:"limit"`
	Offset   int     `json:"offset"`
}

type MemoryToTodoRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Priority    *string `json:"priority"`
	GroupID     *string `json:"group_id"`
}

type MemoryReorderRequest struct {
	Memories []MemoryPosition `json:"memories" binding:"required"`
}

type MemoryPosition struct {
	ID       string `json:"id" binding:"required"`
	Position string `json:"position" binding:"required"`
}

type WebSearchRequest struct {
	Query string `json:"query" binding:"required"`
}

type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type MemoryStats struct {
	Total      int            `json:"total"`
	ByCategory map[string]int `json:"by_category"`
	ThisWeek   int            `json:"this_week"`
	ThisMonth  int            `json:"this_month"`
}

type AIProcessedMemory struct {
	Summary     string   `json:"summary"`
	Category    string   `json:"category"`
	DetectedURL *string  `json:"detected_url"`
	Tags        []string `json:"tags"`
}

type URLSummary struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// MemoryBulkCreateResponse is the response for bulk memory creation from file upload
type MemoryBulkCreateResponse struct {
	Memories     []Memory `json:"memories"`
	TotalCreated int      `json:"total_created"`
	Filename     string   `json:"filename"`
	FileType     string   `json:"file_type"`
}
