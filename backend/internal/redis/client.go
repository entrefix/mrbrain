package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	KeyPrefix = "todomyday:"
)

type Client struct {
	rdb    *redis.Client
	ctx    context.Context
	prefix string
}

type Config struct {
	URL      string
	Password string
	DB       int
	Enabled  bool
}

func NewClient(cfg Config) (*Client, error) {
	if !cfg.Enabled {
		log.Println("Redis is disabled, returning nil client")
		return nil, nil
	}

	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		// If URL parsing fails, try manual configuration
		opt = &redis.Options{
			Addr:     "localhost:6379",
			Password: cfg.Password,
			DB:       cfg.DB,
		}
	} else {
		// Override with explicit password and DB if provided
		if cfg.Password != "" {
			opt.Password = cfg.Password
		}
		if cfg.DB >= 0 {
			opt.DB = cfg.DB
		}
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()

	// Test connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Redis client connected successfully")

	return &Client{
		rdb:    rdb,
		ctx:    ctx,
		prefix: KeyPrefix,
	}, nil
}

func (c *Client) IsEnabled() bool {
	return c != nil && c.rdb != nil
}

func (c *Client) key(key string) string {
	return c.prefix + key
}

func (c *Client) Set(key string, value interface{}, ttl time.Duration) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	
	var val string
	switch v := value.(type) {
	case string:
		val = v
	default:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		val = string(jsonBytes)
	}

	return c.rdb.Set(c.ctx, key, val, ttl).Err()
}

func (c *Client) Get(key string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	val, err := c.rdb.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (c *Client) GetJSON(key string, dest interface{}) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	val, err := c.Get(key)
	if err != nil {
		return err
	}
	if val == "" {
		return redis.Nil
	}

	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) Delete(key string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	return c.rdb.Del(c.ctx, key).Err()
}

func (c *Client) Exists(key string) (bool, error) {
	if !c.IsEnabled() {
		return false, fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	count, err := c.rdb.Exists(c.ctx, key).Result()
	return count > 0, err
}

func (c *Client) Increment(key string) (int64, error) {
	if !c.IsEnabled() {
		return 0, fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	return c.rdb.Incr(c.ctx, key).Result()
}

func (c *Client) SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	if !c.IsEnabled() {
		return false, fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	
	var val string
	switch v := value.(type) {
	case string:
		val = v
	default:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
		val = string(jsonBytes)
	}

	return c.rdb.SetNX(c.ctx, key, val, ttl).Result()
}

func (c *Client) GetSet(key string, value interface{}) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	
	var val string
	switch v := value.(type) {
	case string:
		val = v
	default:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value: %w", err)
		}
		val = string(jsonBytes)
	}

	return c.rdb.GetSet(c.ctx, key, val).Result()
}

func (c *Client) Expire(key string, ttl time.Duration) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	key = c.key(key)
	return c.rdb.Expire(c.ctx, key, ttl).Err()
}

func (c *Client) Keys(pattern string) ([]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Redis is not enabled")
	}

	pattern = c.key(pattern)
	keys, err := c.rdb.Keys(c.ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	// Remove prefix from keys
	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = key[len(c.prefix):]
	}
	return result, nil
}

func (c *Client) DeletePattern(pattern string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	keys, err := c.Keys(pattern)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	// Add prefix back for deletion
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.key(key)
	}

	return c.rdb.Del(c.ctx, fullKeys...).Err()
}

func (c *Client) Ping() error {
	if !c.IsEnabled() {
		return fmt.Errorf("Redis is not enabled")
	}

	return c.rdb.Ping(c.ctx).Err()
}

func (c *Client) Close() error {
	if !c.IsEnabled() {
		return nil
	}

	return c.rdb.Close()
}
