package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/services"
)

func RateLimitMiddleware(redisService *services.RedisService, requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting if Redis is not enabled
		if !redisService.IsEnabled() {
			c.Next()
			return
		}

		userID := GetUserID(c)
		if userID == "" {
			// No user ID, skip rate limiting
			c.Next()
			return
		}

		// Get endpoint path
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Create rate limit key
		key := "rate_limit:" + userID + ":" + endpoint

		// Increment counter with 1 minute expiry
		count, err := redisService.IncrementWithExpiry(key, 1*time.Minute)
		if err != nil {
			// If Redis fails, allow request (graceful degradation)
			c.Next()
			return
		}

		// Check if limit exceeded
		if count > int64(requestsPerMinute) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(requestsPerMinute-int(count)))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(1*time.Minute).Unix(), 10))

		c.Next()
	}
}
