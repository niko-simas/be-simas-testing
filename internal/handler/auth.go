package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"simas-backend/internal/auth"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
	"simas-backend/internal/service"
)

type AuthHandler struct {
	userRepo     *repository.UserRepository
	jwtManager   *auth.JWTManager
	googleOAuth  *auth.GoogleOAuth
	authService  *service.MobileAuthService
	clientID     string
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtManager *auth.JWTManager, googleOAuth *auth.GoogleOAuth, authService *service.MobileAuthService, clientID string) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		jwtManager:  jwtManager,
		googleOAuth: googleOAuth,
		authService: authService,
		clientID:    clientID,
	}
}

type LoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// POST /auth/google
// @Summary Google OAuth login (legacy endpoint)
// @Description Authenticate user using Google ID token. Returns session or pending OAuth state.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body handler.LoginRequest true "Google ID token"
// @Success 200 {object} model.SessionResponse
// @Success 202 {object} model.OAuthPendingResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /auth/google [post]
func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token is required"})
		return
	}

	result, err := h.authService.OAuth(c.Request.Context(), req.IDToken)
	if err != nil {
		respondError(c, err)
		return
	}

	if pending, ok := result.(*model.OAuthPendingResponse); ok {
		c.JSON(http.StatusAccepted, pending)
		return
	}

	c.JSON(http.StatusOK, result)
}

// POST /auth/oauth/complete
// @Summary Complete pending Google OAuth registration
// @Description Complete OAuth registration by providing temp_user_id and WhatsApp number.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body model.CompleteOAuthRequest true "Complete OAuth request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /auth/oauth/complete [post]
func (h *AuthHandler) CompleteOAuth(c *gin.Context) {
	var req model.CompleteOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	session, err := h.authService.CompleteOAuth(c.Request.Context(), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /auth/refresh
// @Summary Refresh access token
// @Description Refresh access token using a valid refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body model.RefreshRequest true "Refresh token request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	session, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_refresh_token"})
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /auth/logout
// @Summary Logout user
// @Description Revoke access token and refresh token to logout user.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body model.RefreshRequest true "Optional refresh token"
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && len(authHeader) >= 8 {
		tokenString := authHeader[7:]
		_ = h.jwtManager.RevokeByToken(tokenString)
	}

	// Attempt to revoke refresh token if provided in body
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		_, recordID, err := h.authService.ValidateRefreshToken(c.Request.Context(), req.RefreshToken)
		if err == nil {
			h.authService.RevokeRefreshToken(c.Request.Context(), recordID, "")
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged_out"})
}

// GET /auth/config — returns public config (google client id)
// @Summary Get public auth config
// @Description Returns public authentication configuration including Google Client ID.
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /auth/config [get]
func (h *AuthHandler) Config(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"google_client_id": h.clientID})
}

// GET /auth/me
// @Summary Get current authenticated user
// @Description Returns the currently authenticated user profile.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.User
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user := userVal.(*model.User)
	c.JSON(http.StatusOK, gin.H{"user": user})
}
