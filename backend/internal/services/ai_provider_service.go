package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/crypto"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

type AIProviderService struct {
	repo      *repository.AIProviderRepository
	encryptor *crypto.Encryptor
}

func NewAIProviderService(repo *repository.AIProviderRepository, encryptor *crypto.Encryptor) *AIProviderService {
	return &AIProviderService{
		repo:      repo,
		encryptor: encryptor,
	}
}

func (s *AIProviderService) Create(userID string, input *models.AIProviderCreate) (*models.AIProvider, error) {
	// Encrypt the API key
	encryptedKey, err := s.encryptor.Encrypt(input.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	// If this should be default, clear other defaults first
	if input.IsDefault {
		if err := s.repo.ClearDefaultForUser(userID); err != nil {
			return nil, err
		}
	}

	provider := &models.AIProvider{
		ID:              uuid.New().String(),
		UserID:          userID,
		Name:            input.Name,
		ProviderType:    input.ProviderType,
		BaseURL:         input.BaseURL,
		APIKeyEncrypted: encryptedKey,
		IsDefault:       input.IsDefault,
		IsEnabled:       true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.Create(provider); err != nil {
		return nil, err
	}

	// Add masked key for response
	provider.APIKeyMasked = crypto.MaskAPIKey(input.APIKey)
	return provider, nil
}

func (s *AIProviderService) GetByID(id, userID string) (*models.AIProvider, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider.UserID != userID {
		return nil, fmt.Errorf("provider not found")
	}

	// Decrypt API key to create masked version
	apiKey, err := s.encryptor.Decrypt(provider.APIKeyEncrypted)
	if err == nil {
		provider.APIKeyMasked = crypto.MaskAPIKey(apiKey)
	}
	return provider, nil
}

func (s *AIProviderService) GetByUserID(userID string) ([]models.AIProvider, error) {
	providers, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Add masked keys
	for i := range providers {
		apiKey, err := s.encryptor.Decrypt(providers[i].APIKeyEncrypted)
		if err == nil {
			providers[i].APIKeyMasked = crypto.MaskAPIKey(apiKey)
		}
	}
	return providers, nil
}

func (s *AIProviderService) GetDefaultByUserID(userID string) (*models.AIProvider, error) {
	provider, err := s.repo.GetDefaultByUserID(userID)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *AIProviderService) Update(id, userID string, input *models.AIProviderUpdate) (*models.AIProvider, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider.UserID != userID {
		return nil, fmt.Errorf("provider not found")
	}

	if input.Name != nil {
		provider.Name = *input.Name
	}
	if input.BaseURL != nil {
		provider.BaseURL = *input.BaseURL
	}
	if input.APIKey != nil {
		encryptedKey, err := s.encryptor.Encrypt(*input.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		provider.APIKeyEncrypted = encryptedKey
	}
	if input.SelectedModel != nil {
		provider.SelectedModel = input.SelectedModel
	}
	if input.IsDefault != nil && *input.IsDefault {
		if err := s.repo.ClearDefaultForUser(userID); err != nil {
			return nil, err
		}
		provider.IsDefault = true
	} else if input.IsDefault != nil {
		provider.IsDefault = *input.IsDefault
	}
	if input.IsEnabled != nil {
		provider.IsEnabled = *input.IsEnabled
	}

	if err := s.repo.Update(provider); err != nil {
		return nil, err
	}

	// Add masked key for response
	apiKey, err := s.encryptor.Decrypt(provider.APIKeyEncrypted)
	if err == nil {
		provider.APIKeyMasked = crypto.MaskAPIKey(apiKey)
	}
	return provider, nil
}

func (s *AIProviderService) Delete(id, userID string) error {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if provider.UserID != userID {
		return fmt.Errorf("provider not found")
	}
	return s.repo.Delete(id)
}

func (s *AIProviderService) TestConnection(input *models.TestConnectionRequest) (*models.TestConnectionResponse, error) {
	switch input.ProviderType {
	case models.ProviderTypeOpenAI, models.ProviderTypeCustom:
		return s.testOpenAICompatible(input.BaseURL, input.APIKey)
	case models.ProviderTypeAnthropic:
		return s.testAnthropic(input.BaseURL, input.APIKey)
	case models.ProviderTypeGoogle:
		return s.testGoogle(input.BaseURL, input.APIKey)
	default:
		return &models.TestConnectionResponse{
			Success: false,
			Message: "Unknown provider type",
		}, nil
	}
}

func (s *AIProviderService) testOpenAICompatible(baseURL, apiKey string) (*models.TestConnectionResponse, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return &models.TestConnectionResponse{
			Success: false,
			Message: "Invalid API key",
		}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	modelIDs := make([]string, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		// Filter for chat models
		if strings.Contains(m.ID, "gpt") || strings.Contains(m.ID, "claude") || strings.Contains(m.ID, "llama") || strings.Contains(m.ID, "mistral") {
			modelIDs = append(modelIDs, m.ID)
		}
	}

	// If no chat models found, include all models
	if len(modelIDs) == 0 {
		for _, m := range modelsResp.Data {
			modelIDs = append(modelIDs, m.ID)
		}
	}

	return &models.TestConnectionResponse{
		Success: true,
		Message: "Connection successful",
		Models:  modelIDs,
	}, nil
}

func (s *AIProviderService) testAnthropic(baseURL, apiKey string) (*models.TestConnectionResponse, error) {
	// Anthropic doesn't have a models endpoint, so we test with a minimal message
	url := strings.TrimSuffix(baseURL, "/") + "/messages"

	body := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return &models.TestConnectionResponse{
			Success: false,
			Message: "Invalid API key",
		}, nil
	}

	// Anthropic returns known models
	knownModels := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	return &models.TestConnectionResponse{
		Success: true,
		Message: "Connection successful",
		Models:  knownModels,
	}, nil
}

func (s *AIProviderService) testGoogle(baseURL, apiKey string) (*models.TestConnectionResponse, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/models?key=" + apiKey

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return &models.TestConnectionResponse{
			Success: false,
			Message: "Invalid API key",
		}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	var modelsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	modelIDs := make([]string, 0, len(modelsResp.Models))
	for _, m := range modelsResp.Models {
		// Extract model ID from name (format: models/model-id)
		parts := strings.Split(m.Name, "/")
		if len(parts) > 1 && strings.Contains(parts[1], "gemini") {
			modelIDs = append(modelIDs, parts[1])
		}
	}

	return &models.TestConnectionResponse{
		Success: true,
		Message: "Connection successful",
		Models:  modelIDs,
	}, nil
}

func (s *AIProviderService) FetchAndSaveModels(id, userID string) ([]models.AIProviderModel, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider.UserID != userID {
		return nil, fmt.Errorf("provider not found")
	}

	// Decrypt API key
	apiKey, err := s.encryptor.Decrypt(provider.APIKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// Test connection to get models
	testResult, err := s.TestConnection(&models.TestConnectionRequest{
		ProviderType: provider.ProviderType,
		BaseURL:      provider.BaseURL,
		APIKey:       apiKey,
	})
	if err != nil {
		return nil, err
	}
	if !testResult.Success {
		return nil, fmt.Errorf("failed to fetch models: %s", testResult.Message)
	}

	// Convert to AIProviderModel
	providerModels := make([]models.AIProviderModel, len(testResult.Models))
	for i, modelID := range testResult.Models {
		providerModels[i] = models.AIProviderModel{
			ID:         uuid.New().String(),
			ProviderID: provider.ID,
			ModelID:    modelID,
			ModelName:  modelID,
			CreatedAt:  time.Now(),
		}
	}

	// Save models
	if err := s.repo.SaveModels(provider.ID, providerModels); err != nil {
		return nil, err
	}

	return providerModels, nil
}

func (s *AIProviderService) GetModels(id, userID string) ([]models.AIProviderModel, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider.UserID != userID {
		return nil, fmt.Errorf("provider not found")
	}

	return s.repo.GetModelsByProviderID(id)
}

// GetDecryptedAPIKey returns the decrypted API key for a provider
func (s *AIProviderService) GetDecryptedAPIKey(provider *models.AIProvider) (string, error) {
	return s.encryptor.Decrypt(provider.APIKeyEncrypted)
}
