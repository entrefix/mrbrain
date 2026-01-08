package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/todomyday/backend/internal/services"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const UserIDKey = "userID"

func AuthMiddleware(supabaseAuthService *services.SupabaseAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Check for Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}
		tokenString := parts[1]

		// Verify Supabase JWT token
		claims, err := supabaseAuthService.VerifyToken(tokenString)
		if err != nil {
			// Log the error for debugging
			fmt.Printf("DEBUG: Token verification failed: %v\n", err)
			fmt.Printf("DEBUG: Token (first 50 chars): %s\n", tokenString[:min(50, len(tokenString))])
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token", "details": err.Error()})
			c.Abort()
			return
		}

		// Sync user from Supabase to local database
		user, err := supabaseAuthService.SyncUserFromToken(claims)
		if err != nil {
			fmt.Printf("ERROR: Failed to sync user from token: %v\n", err)
			fmt.Printf("ERROR: Claims - Sub: %s, Email: %s\n", claims.Sub, claims.Email)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sync user", "details": err.Error()})
			c.Abort()
			return
		}

		// Set user ID (local DB ID) in context for downstream handlers
		c.Set(UserIDKey, user.ID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}
