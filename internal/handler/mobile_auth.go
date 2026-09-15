package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"simas-backend/internal/auth"
	"simas-backend/internal/model"
	"simas-backend/internal/service"
)

type MobileAuthHandler struct {
	authService *service.MobileAuthService
}

func NewMobileAuthHandler(authService *service.MobileAuthService) *MobileAuthHandler {
	return &MobileAuthHandler{authService: authService}
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: "email_already_exists", Message: "Email sudah terdaftar"})
	case errors.Is(err, service.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "invalid_credentials", Message: "Email atau password salah"})
	case errors.Is(err, service.ErrEmailNotVerified):
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: "email_not_verified", Message: "Email belum diverifikasi, cek kode OTP"})
	case errors.Is(err, service.ErrInvalidOTP):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_otp", Message: "Kode OTP salah atau sudah kedaluwarsa"})
	case errors.Is(err, service.ErrResendCooldown):
		c.JSON(http.StatusTooManyRequests, model.ErrorResponse{Code: "resend_cooldown", Message: "Tunggu sebelum mengirim ulang OTP"})
	case errors.Is(err, service.ErrPendingOAuth):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_session", Message: "Sesi tidak valid atau sudah kedaluwarsa"})
	case errors.Is(err, service.ErrInvalidWhatsApp):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_whatsapp", Message: "Nomor WhatsApp tidak valid"})
	case errors.Is(err, service.ErrGoogleTokenInvalid):
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "invalid_google_token", Message: "Token Google tidak valid"})
	case errors.Is(err, auth.ErrWeakPassword):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "weak_password", Message: "Password minimal 8 karakter, harus ada huruf besar, huruf kecil, angka, dan karakter khusus"})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: "Terjadi kesalahan server"})
	}
}

// POST /api/v1/auth/mobile/register
// @Summary Register new mobile user
// @Description Register a new user with email, WhatsApp, and password. An OTP will be sent to the email.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "Registration request"
// @Success 201 {object} model.RegisterResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/auth/mobile/register [post]
func (h *MobileAuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	if err := h.authService.Register(c.Request.Context(), &req); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, model.RegisterResponse{
		Message:   "verification_code_sent",
		Email:     req.Email,
		ExpiresIn: 180,
	})
}

// POST /api/v1/auth/mobile/verify-otp
// @Summary Verify OTP code
// @Description Verify the OTP code sent to the user's email during registration.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.VerifyOTPRequest true "OTP verification request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /api/v1/auth/mobile/verify-otp [post]
func (h *MobileAuthHandler) VerifyOTP(c *gin.Context) {
	var req model.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	session, err := h.authService.VerifyOTP(c.Request.Context(), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /api/v1/auth/mobile/resend-otp
// @Summary Resend OTP code
// @Description Resend OTP code to the user's email.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.ResendOTPRequest true "Resend OTP request"
// @Success 200 {object} model.ResendResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 429 {object} model.ErrorResponse
// @Router /api/v1/auth/mobile/resend-otp [post]
func (h *MobileAuthHandler) ResendOTP(c *gin.Context) {
	var req model.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	if err := h.authService.ResendOTP(c.Request.Context(), &req); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResendResponse{
		Message:   "verification_code_sent",
		ExpiresIn: 180,
	})
}

// POST /api/v1/auth/mobile/login
// @Summary Mobile user login
// @Description Login with email and password for mobile users.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.MobileLoginRequest true "Login request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/auth/mobile/login [post]
func (h *MobileAuthHandler) Login(c *gin.Context) {
	var req model.MobileLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	session, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /api/v1/auth/mobile/oauth
// @Summary Google OAuth login for mobile
// @Description Authenticate mobile user using Google token. Returns session or pending state if new user.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.MobileOAuthRequest true "Google OAuth request"
// @Success 200 {object} model.SessionResponse
// @Success 202 {object} model.OAuthPendingResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /api/v1/auth/mobile/oauth [post]
func (h *MobileAuthHandler) OAuth(c *gin.Context) {
	var req model.MobileOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	result, err := h.authService.OAuth(c.Request.Context(), req.GoogleToken)
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

// POST /api/v1/auth/mobile/oauth/complete
// @Summary Complete mobile OAuth registration
// @Description Complete Google OAuth registration for mobile by providing temp_user_id and WhatsApp number.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.CompleteOAuthRequest true "Complete OAuth request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/auth/mobile/oauth/complete [post]
func (h *MobileAuthHandler) CompleteOAuth(c *gin.Context) {
	var req model.CompleteOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	session, err := h.authService.CompleteOAuth(c.Request.Context(), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// POST /api/v1/auth/refresh
// @Summary Refresh access token
// @Description Refresh access token using a valid refresh token.
// @Tags Mobile Auth
// @Accept json
// @Produce json
// @Param request body model.RefreshRequest true "Refresh token request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *MobileAuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}
	session, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

// GET /api/v1/mobile/status-verifikasi
// @Summary Get mobile user verification status
// @Description Returns the verification and account status for the authenticated mobile user.
// @Tags Mobile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.StatusVerifikasiResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /api/v1/mobile/status-verifikasi [get]
func (h *MobileAuthHandler) StatusVerifikasi(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Unauthorized"})
		return
	}
	user := userVal.(*model.User)
	c.JSON(http.StatusOK, h.authService.GetStatus(user))
}
