package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/services"
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

func HealthCheckWithRedis(redisService *services.RedisService) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"status": "healthy",
		}

		// Check Redis status
		if redisService != nil {
			redisStatus := "disabled"
			if redisService.IsEnabled() {
				if err := redisService.Ping(); err == nil {
					redisStatus = "connected"
				} else {
					redisStatus = "disconnected"
					status["status"] = "degraded"
				}
			}
			status["redis"] = redisStatus
		} else {
			status["redis"] = "not_configured"
		}

		httpStatus := http.StatusOK
		if status["status"] == "degraded" {
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, status)
	}
}
