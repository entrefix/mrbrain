package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/crypto"
	"github.com/todomyday/backend/internal/middleware"
	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
	"github.com/todomyday/backend/internal/services"
)

type CalendarHandler struct {
	config          *CalendarHandlerConfig
	oauthStateService *services.OAuthStateService
}

type CalendarHandlerConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	RedirectURI        string
	Encryptor          *crypto.Encryptor
	CalendarRepo       *repository.CalendarRepository
}

func NewCalendarHandler(config *CalendarHandlerConfig, oauthStateService *services.OAuthStateService) *CalendarHandler {
	return &CalendarHandler{
		config:          config,
		oauthStateService: oauthStateService,
	}
}

// InitiateOAuth starts the Google Calendar OAuth flow
func (h *CalendarHandler) InitiateOAuth(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Generate state token
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store state in Redis with user ID
	err := h.oauthStateService.StoreState(state, services.OAuthStateData{
		UserID:   userID,
		Provider: "google",
		ExtraData: map[string]interface{}{
			"redirect_uri": h.config.RedirectURI,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store state"})
		return
	}

	// Build Google OAuth URL
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent&state=%s",
		h.config.GoogleClientID,
		h.config.RedirectURI,
		"https://www.googleapis.com/auth/calendar",
		state,
	)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

// OAuthCallback handles the OAuth callback from Google
func (h *CalendarHandler) OAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=%s", h.config.RedirectURI, errorParam))
		return
	}

	if code == "" || state == "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=missing_code_or_state", h.config.RedirectURI))
		return
	}

	// Validate and retrieve state
	stateData, err := h.oauthStateService.GetState(state)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=invalid_state", h.config.RedirectURI))
		return
	}

	userID := stateData.UserID

	// Exchange code for tokens
	tokenURL := "https://oauth2.googleapis.com/token"
	reqBody := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=%s",
		h.config.GoogleClientID,
		h.config.GoogleClientSecret,
		code,
		h.config.RedirectURI,
	)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded",
		strings.NewReader(reqBody))
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=token_exchange_failed", h.config.RedirectURI))
		return
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=token_decode_failed", h.config.RedirectURI))
		return
	}

	// Get user's calendar info
	calendarInfo, err := h.getCalendarInfo(tokenResp.AccessToken)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=calendar_info_failed", h.config.RedirectURI))
		return
	}

	// Encrypt tokens
	encryptedAccessToken, err := h.config.Encryptor.Encrypt(tokenResp.AccessToken)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=encryption_failed", h.config.RedirectURI))
		return
	}

	var encryptedRefreshToken string
	if tokenResp.RefreshToken != "" {
		encryptedRefreshToken, err = h.config.Encryptor.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=encryption_failed", h.config.RedirectURI))
			return
		}
	}

	// Calculate token expiration
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Check if connection exists
	exists, err := h.config.CalendarRepo.Exists(userID)
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=database_error", h.config.RedirectURI))
		return
	}

	conn := &models.CalendarConnection{
		ID:             uuid.New().String(),
		UserID:         userID,
		Provider:       "google",
		AccessToken:    encryptedAccessToken,
		RefreshToken:   encryptedRefreshToken,
		TokenExpiresAt: &expiresAt,
		CalendarID:     &calendarInfo.ID,
		CalendarEmail:  &calendarInfo.Email,
		IsEnabled:      true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if exists {
		// Update existing connection
		existingConn, err := h.config.CalendarRepo.GetByUserID(userID)
		if err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=database_error", h.config.RedirectURI))
			return
		}
		if existingConn != nil {
			conn.ID = existingConn.ID
			conn.CreatedAt = existingConn.CreatedAt
			if err := h.config.CalendarRepo.Update(conn); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=update_failed", h.config.RedirectURI))
				return
			}
		} else {
			// Create if somehow doesn't exist
			if err := h.config.CalendarRepo.Create(conn); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=create_failed", h.config.RedirectURI))
				return
			}
		}
	} else {
		// Create new connection
		if err := h.config.CalendarRepo.Create(conn); err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s?error=create_failed", h.config.RedirectURI))
			return
		}
	}

	// Redirect to frontend with success
	c.Redirect(http.StatusFound, fmt.Sprintf("%s?success=true", h.config.RedirectURI))
}

// GetStatus returns the current calendar connection status
func (h *CalendarHandler) GetStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := h.config.CalendarRepo.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database_error"})
		return
	}

	if conn == nil {
		c.JSON(http.StatusOK, gin.H{
			"connected": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected":     true,
		"calendar_email": conn.CalendarEmail,
		"is_enabled":     conn.IsEnabled,
	})
}

// Disconnect removes the calendar connection
func (h *CalendarHandler) Disconnect(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.config.CalendarRepo.Delete(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disconnected"})
}

// Helper function to get calendar info
func (h *CalendarHandler) getCalendarInfo(accessToken string) (*CalendarInfo, error) {
	// Get user's email from userinfo first
	userInfoReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	userInfoReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	userInfoResp, err := http.DefaultClient.Do(userInfoReq)
	if err != nil {
		return nil, err
	}
	defer userInfoResp.Body.Close()

	if userInfoResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userInfoResp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// Get primary calendar
	req, err := http.NewRequest("GET", "https://www.googleapis.com/calendar/v3/users/me/calendarList/primary", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get calendar: %s", string(body))
	}

	var calendar struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&calendar); err != nil {
		return nil, err
	}

	return &CalendarInfo{
		ID:    calendar.ID,
		Email: userInfo.Email,
	}, nil
}

type CalendarInfo struct {
	ID    string
	Email string
}
