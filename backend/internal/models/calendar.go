package models

import "time"

type CalendarConnection struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Provider          string     `json:"provider"` // "google"
	AccessToken       string     `json:"-"`       // Encrypted, not returned in JSON
	RefreshToken      string     `json:"-"`       // Encrypted, not returned in JSON
	TokenExpiresAt    *time.Time `json:"token_expires_at"`
	CalendarID        *string    `json:"calendar_id"`
	CalendarEmail     *string    `json:"calendar_email"`
	IsEnabled         bool       `json:"is_enabled"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
