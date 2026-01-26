package services

import (
	"fmt"
	"time"
)

type OAuthStateService struct {
	redisService *RedisService
	ttl          time.Duration
}

type OAuthStateData struct {
	UserID    string                 `json:"user_id"`
	Provider  string                 `json:"provider"`
	ExtraData map[string]interface{} `json:"extra_data,omitempty"`
}

func NewOAuthStateService(redisService *RedisService, ttl time.Duration) *OAuthStateService {
	if ttl == 0 {
		ttl = 10 * time.Minute // Default TTL
	}
	return &OAuthStateService{
		redisService: redisService,
		ttl:          ttl,
	}
}

func (s *OAuthStateService) StoreState(state string, data OAuthStateData) error {
	if !s.redisService.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	key := fmt.Sprintf("oauth:state:%s", state)
	return s.redisService.SetJSON(key, data, s.ttl)
}

func (s *OAuthStateService) GetState(state string) (*OAuthStateData, error) {
	if !s.redisService.IsEnabled() {
		return nil, fmt.Errorf("Redis is not enabled")
	}

	key := fmt.Sprintf("oauth:state:%s", state)
	var data OAuthStateData
	err := s.redisService.GetJSON(key, &data)
	if err != nil {
		return nil, fmt.Errorf("state not found or expired: %w", err)
	}

	// Delete state after retrieval (one-time use)
	_ = s.redisService.Delete(key)

	return &data, nil
}

func (s *OAuthStateService) ValidateState(state string) (bool, error) {
	if !s.redisService.IsEnabled() {
		return false, fmt.Errorf("Redis is not enabled")
	}

	key := fmt.Sprintf("oauth:state:%s", state)
	exists, err := s.redisService.Exists(key)
	return exists, err
}

func (s *OAuthStateService) DeleteState(state string) error {
	if !s.redisService.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	key := fmt.Sprintf("oauth:state:%s", state)
	return s.redisService.Delete(key)
}

// Helper to create state with JSON data
func (s *OAuthStateService) StoreStateWithData(state string, userID, provider string, extraData map[string]interface{}) error {
	data := OAuthStateData{
		UserID:    userID,
		Provider:  provider,
		ExtraData: extraData,
	}
	return s.StoreState(state, data)
}
