package repository

import (
	"database/sql"
	"time"

	"github.com/todomyday/backend/internal/models"
)

type AIProviderRepository struct {
	db *sql.DB
}

func NewAIProviderRepository(db *sql.DB) *AIProviderRepository {
	return &AIProviderRepository{db: db}
}

func (r *AIProviderRepository) Create(provider *models.AIProvider) error {
	query := `
		INSERT INTO ai_providers (id, user_id, name, provider_type, base_url, api_key_encrypted, selected_model, is_default, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		provider.ID,
		provider.UserID,
		provider.Name,
		provider.ProviderType,
		provider.BaseURL,
		provider.APIKeyEncrypted,
		provider.SelectedModel,
		provider.IsDefault,
		provider.IsEnabled,
		provider.CreatedAt,
		provider.UpdatedAt,
	)
	return err
}

func (r *AIProviderRepository) GetByID(id string) (*models.AIProvider, error) {
	query := `
		SELECT id, user_id, name, provider_type, base_url, api_key_encrypted, selected_model, is_default, is_enabled, created_at, updated_at
		FROM ai_providers WHERE id = ?
	`
	var provider models.AIProvider
	var selectedModel sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&provider.ID,
		&provider.UserID,
		&provider.Name,
		&provider.ProviderType,
		&provider.BaseURL,
		&provider.APIKeyEncrypted,
		&selectedModel,
		&provider.IsDefault,
		&provider.IsEnabled,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if selectedModel.Valid {
		provider.SelectedModel = &selectedModel.String
	}
	return &provider, nil
}

func (r *AIProviderRepository) GetByUserID(userID string) ([]models.AIProvider, error) {
	query := `
		SELECT id, user_id, name, provider_type, base_url, api_key_encrypted, selected_model, is_default, is_enabled, created_at, updated_at
		FROM ai_providers WHERE user_id = ? ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []models.AIProvider
	for rows.Next() {
		var provider models.AIProvider
		var selectedModel sql.NullString
		if err := rows.Scan(
			&provider.ID,
			&provider.UserID,
			&provider.Name,
			&provider.ProviderType,
			&provider.BaseURL,
			&provider.APIKeyEncrypted,
			&selectedModel,
			&provider.IsDefault,
			&provider.IsEnabled,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if selectedModel.Valid {
			provider.SelectedModel = &selectedModel.String
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func (r *AIProviderRepository) GetDefaultByUserID(userID string) (*models.AIProvider, error) {
	query := `
		SELECT id, user_id, name, provider_type, base_url, api_key_encrypted, selected_model, is_default, is_enabled, created_at, updated_at
		FROM ai_providers WHERE user_id = ? AND is_default = 1 AND is_enabled = 1 LIMIT 1
	`
	var provider models.AIProvider
	var selectedModel sql.NullString
	err := r.db.QueryRow(query, userID).Scan(
		&provider.ID,
		&provider.UserID,
		&provider.Name,
		&provider.ProviderType,
		&provider.BaseURL,
		&provider.APIKeyEncrypted,
		&selectedModel,
		&provider.IsDefault,
		&provider.IsEnabled,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if selectedModel.Valid {
		provider.SelectedModel = &selectedModel.String
	}
	return &provider, nil
}

func (r *AIProviderRepository) Update(provider *models.AIProvider) error {
	query := `
		UPDATE ai_providers
		SET name = ?, base_url = ?, api_key_encrypted = ?, selected_model = ?, is_default = ?, is_enabled = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		provider.Name,
		provider.BaseURL,
		provider.APIKeyEncrypted,
		provider.SelectedModel,
		provider.IsDefault,
		provider.IsEnabled,
		time.Now(),
		provider.ID,
	)
	return err
}

func (r *AIProviderRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM ai_providers WHERE id = ?", id)
	return err
}

func (r *AIProviderRepository) ClearDefaultForUser(userID string) error {
	_, err := r.db.Exec("UPDATE ai_providers SET is_default = 0 WHERE user_id = ?", userID)
	return err
}

// Model methods
func (r *AIProviderRepository) SaveModels(providerID string, models []models.AIProviderModel) error {
	// Delete existing models
	if _, err := r.db.Exec("DELETE FROM ai_provider_models WHERE provider_id = ?", providerID); err != nil {
		return err
	}

	// Insert new models
	for _, model := range models {
		query := `INSERT INTO ai_provider_models (id, provider_id, model_id, model_name, created_at) VALUES (?, ?, ?, ?, ?)`
		if _, err := r.db.Exec(query, model.ID, model.ProviderID, model.ModelID, model.ModelName, model.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *AIProviderRepository) GetModelsByProviderID(providerID string) ([]models.AIProviderModel, error) {
	query := `SELECT id, provider_id, model_id, model_name, created_at FROM ai_provider_models WHERE provider_id = ? ORDER BY model_name`
	rows, err := r.db.Query(query, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providerModels []models.AIProviderModel
	for rows.Next() {
		var model models.AIProviderModel
		if err := rows.Scan(&model.ID, &model.ProviderID, &model.ModelID, &model.ModelName, &model.CreatedAt); err != nil {
			return nil, err
		}
		providerModels = append(providerModels, model)
	}
	return providerModels, nil
}
