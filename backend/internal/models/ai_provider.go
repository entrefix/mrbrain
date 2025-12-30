package models

import "time"

type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeGoogle    ProviderType = "google"
	ProviderTypeCustom    ProviderType = "custom"
)

type AIProvider struct {
	ID              string       `json:"id"`
	UserID          string       `json:"user_id"`
	Name            string       `json:"name"`
	ProviderType    ProviderType `json:"provider_type"`
	BaseURL         string       `json:"base_url"`
	APIKeyEncrypted string       `json:"-"` // Never expose encrypted key
	APIKeyMasked    string       `json:"api_key_masked,omitempty"`
	SelectedModel   *string      `json:"selected_model"`
	IsDefault       bool         `json:"is_default"`
	IsEnabled       bool         `json:"is_enabled"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type AIProviderModel struct {
	ID         string    `json:"id"`
	ProviderID string    `json:"provider_id"`
	ModelID    string    `json:"model_id"`
	ModelName  string    `json:"model_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type AIProviderCreate struct {
	Name         string       `json:"name" binding:"required"`
	ProviderType ProviderType `json:"provider_type" binding:"required"`
	BaseURL      string       `json:"base_url" binding:"required"`
	APIKey       string       `json:"api_key" binding:"required"`
	IsDefault    bool         `json:"is_default"`
}

type AIProviderUpdate struct {
	Name          *string `json:"name"`
	BaseURL       *string `json:"base_url"`
	APIKey        *string `json:"api_key"`
	SelectedModel *string `json:"selected_model"`
	IsDefault     *bool   `json:"is_default"`
	IsEnabled     *bool   `json:"is_enabled"`
}

type TestConnectionRequest struct {
	ProviderType ProviderType `json:"provider_type" binding:"required"`
	BaseURL      string       `json:"base_url" binding:"required"`
	APIKey       string       `json:"api_key" binding:"required"`
}

type TestConnectionResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Models  []string `json:"models,omitempty"`
}

// GetDefaultBaseURL returns the default base URL for a provider type
func GetDefaultBaseURL(providerType ProviderType) string {
	switch providerType {
	case ProviderTypeOpenAI:
		return "https://api.openai.com/v1"
	case ProviderTypeAnthropic:
		return "https://api.anthropic.com/v1"
	case ProviderTypeGoogle:
		return "https://generativelanguage.googleapis.com/v1beta"
	default:
		return ""
	}
}
