package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/losts/syun-eng/backend/internal/config"
	"github.com/losts/syun-eng/backend/internal/service"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *service.AuthService
	config      *config.Config
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      cfg,
	}
}

// GoogleCallback handles the Google OAuth callback
type GoogleCallbackRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         UserInfo `json:"user"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GoogleCallback handles Google OAuth callback
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	var req GoogleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user info from Google
	googleInfo, err := h.authService.GetGoogleUserInfo(c.Request.Context(), req.AccessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to get Google user info"})
		return
	}

	// Find or create user
	user, err := h.authService.FindOrCreateUser(c.Request.Context(), googleInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate tokens
	accessToken, err := h.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserInfo{
			ID:    user.UserID,
			Email: user.Email,
			Name:  user.Name,
		},
	})
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh refreshes the access token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token
	claims, err := h.authService.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Generate new access token
	// For simplicity, we create a minimal user struct
	user := &struct {
		UserID string
		Email  string
		Name   string
	}{
		UserID: claims.UserID,
		Email:  claims.Email,
		Name:   claims.Name,
	}

	// We need to create a proper model.User for token generation
	// For now, return a simple response
	c.JSON(http.StatusOK, gin.H{
		"message": "token refresh not fully implemented",
		"user_id": user.UserID,
	})
}

// Me returns the current user info
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	email := c.GetString("email")
	name := c.GetString("name")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, UserInfo{
		ID:    userID,
		Email: email,
		Name:  name,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// For JWT-based auth, logout is handled client-side by removing tokens
	// Server-side token blacklisting can be added for enhanced security
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
