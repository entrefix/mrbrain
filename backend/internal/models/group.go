package models

import "time"

type Group struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"user_id"`
	Name      string    `json:"name"`
	ColorCode string    `json:"color_code"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GroupCreateRequest struct {
	Name      string `json:"name" binding:"required"`
	ColorCode string `json:"color_code"`
}

type GroupUpdateRequest struct {
	Name      *string `json:"name"`
	ColorCode *string `json:"color_code"`
}
