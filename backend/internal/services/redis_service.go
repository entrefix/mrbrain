package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/todomyday/backend/internal/redis"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(client *redis.Client) *RedisService {
	return &RedisService{
		client: client,
	}
}

func (s *RedisService) IsEnabled() bool {
	return s.client != nil && s.client.IsEnabled()
}

func (s *RedisService) Set(key string, value interface{}, ttl time.Duration) error {
	if !s.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return s.client.Set(key, value, ttl)
}

func (s *RedisService) Get(key string) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("Redis is not enabled")
	}
	return s.client.Get(key)
}

func (s *RedisService) GetJSON(key string, dest interface{}) error {
	if !s.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return s.client.GetJSON(key, dest)
}

func (s *RedisService) Delete(key string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return s.client.Delete(key)
}

func (s *RedisService) Exists(key string) (bool, error) {
	if !s.IsEnabled() {
		return false, fmt.Errorf("Redis is not enabled")
	}
	return s.client.Exists(key)
}

func (s *RedisService) Increment(key string) (int64, error) {
	if !s.IsEnabled() {
		return 0, fmt.Errorf("Redis is not enabled")
	}
	return s.client.Increment(key)
}

func (s *RedisService) SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	if !s.IsEnabled() {
		return false, fmt.Errorf("Redis is not enabled")
	}
	return s.client.SetNX(key, value, ttl)
}

func (s *RedisService) GetSet(key string, value interface{}) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("Redis is not enabled")
	}
	return s.client.GetSet(key, value)
}

func (s *RedisService) Expire(key string, ttl time.Duration) error {
	if !s.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return s.client.Expire(key, ttl)
}

func (s *RedisService) Keys(pattern string) ([]string, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("Redis is not enabled")
	}
	return s.client.Keys(pattern)
}

func (s *RedisService) DeletePattern(pattern string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return s.client.DeletePattern(pattern)
}

func (s *RedisService) Ping() error {
	if !s.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}
	return s.client.Ping()
}

// Helper methods for common patterns

func (s *RedisService) SetJSON(key string, value interface{}, ttl time.Duration) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return s.Set(key, string(jsonBytes), ttl)
}

func (s *RedisService) GetAndDelete(key string) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("Redis is not enabled")
	}

	val, err := s.Get(key)
	if err != nil {
		return "", err
	}

	if val != "" {
		if err := s.Delete(key); err != nil {
			return val, err
		}
	}

	return val, nil
}

func (s *RedisService) IncrementWithExpiry(key string, ttl time.Duration) (int64, error) {
	count, err := s.Increment(key)
	if err != nil {
		return 0, err
	}

	// Set expiry on first increment
	if count == 1 {
		if err := s.Expire(key, ttl); err != nil {
			return count, err
		}
	}

	return count, nil
}
