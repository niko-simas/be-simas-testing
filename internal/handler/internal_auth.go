package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"simas-backend/internal/auth"
	"simas-backend/internal/model"
	"simas-backend/internal/service"
)

type InternalAuthHandler struct {
	internalAuthService *service.InternalAuthService
}

func NewInternalAuthHandler(internalAuthService *service.InternalAuthService) *InternalAuthHandler {
	return &InternalAuthHandler{internalAuthService: internalAuthService}
}

type RegisterSuperAdminRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=3,max=255"`
	Password string `json:"password" binding:"required,min=8"`
}

type PusatLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type InternalLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ForceUpdateRequest struct {
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8"`
}

func respondInternalError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInternalInvalidCredentials):
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "invalid_credentials", Message: "Email atau password salah"})
	case errors.Is(err, service.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: "email_already_exists", Message: "Email sudah terdaftar"})
	case errors.Is(err, service.ErrForbiddenRole):
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: "forbidden", Message: "Tidak memiliki izin untuk membuat super_admin"})
	case errors.Is(err, service.ErrEmailNotVerified):
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: "account_inactive", Message: "Akun belum aktif"})
	case errors.Is(err, service.ErrPasswordMismatch):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "password_mismatch", Message: "Konfirmasi password tidak cocok"})
	case errors.Is(err, service.ErrSamePassword):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "same_password", Message: "Password baru harus berbeda dari password lama"})
	case errors.Is(err, service.ErrInvalidTempToken):
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "invalid_temp_token", Message: "Token tidak valid atau sudah kedaluwarsa"})
	case errors.Is(err, service.ErrNotInternalAccount):
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: "not_internal", Message: "Akun tidak memiliki akses untuk update password"})
	case errors.Is(err, auth.ErrWeakPassword):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "weak_password", Message: "Password minimal 8 karakter, harus ada huruf besar, huruf kecil, angka, dan karakter khusus"})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: "Terjadi kesalahan server"})
	}
}

// POST /auth/pusat/login
// @Summary Pusat (Super Admin) login
// @Description Login for super admin / pusat accounts.
// @Tags Internal Auth
// @Accept json
// @Produce json
// @Param request body handler.PusatLoginRequest true "Pusat login request"
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Router /auth/pusat/login [post]
func (h *InternalAuthHandler) PusatLogin(c *gin.Context) {
	var req PusatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	session, err := h.internalAuthService.PusatLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, session)
}

// POST /auth/pusat/register — protected by RequireAuth + RequireRole("super_admin")
// @Summary Register new super admin
// @Description Register a new super admin account. Only accessible by existing super_admin.
// @Tags Internal Auth
// @Accept json
// @Produce json
// @Param request body handler.RegisterSuperAdminRequest true "Register super admin request"
// @Security BearerAuth
// @Success 201 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse
// @Router /auth/pusat/register [post]
func (h *InternalAuthHandler) RegisterSuperAdmin(c *gin.Context) {
	var req RegisterSuperAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Unauthorized"})
		return
	}
	creator := userVal.(*model.User)

	session, err := h.internalAuthService.RegisterSuperAdmin(c.Request.Context(), req.Email, req.Name, req.Password, creator.Role)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, session)
}

// POST /auth/internal/login
// @Summary Internal user login
// @Description Login for internal users (staff, teachers, etc.).
// @Tags Internal Auth
// @Accept json
// @Produce json
// @Param request body handler.InternalLoginRequest true "Internal login request"
// @Success 200 {object} model.SessionResponse
// @Success 202 {object} model.ForceUpdateResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Router /auth/internal/login [post]
func (h *InternalAuthHandler) InternalLogin(c *gin.Context) {
	var req InternalLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	result, err := h.internalAuthService.InternalLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if forceUpdate, ok := result.(*model.ForceUpdateResponse); ok {
		c.JSON(http.StatusAccepted, forceUpdate)
		return
	}

	c.JSON(http.StatusOK, result)
}

// PUT /auth/internal/update-password
// @Summary Force update internal user password
// @Description Update password for internal user on first login or forced reset.
// @Tags Internal Auth
// @Accept json
// @Produce json
// @Param request body handler.ForceUpdateRequest true "Force update password request"
// @Security BearerAuth
// @Success 200 {object} model.SessionResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Router /auth/internal/update-password [put]
func (h *InternalAuthHandler) ForceUpdatePassword(c *gin.Context) {
	var req ForceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	// Get temp token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "missing_token", Message: "Token wajib disertakan"})
		return
	}
	tempToken := authHeader[7:] // strip "Bearer "

	session, err := h.internalAuthService.ForceUpdatePassword(c.Request.Context(), tempToken, req.OldPassword, req.NewPassword, req.ConfirmPassword)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, session)
}
