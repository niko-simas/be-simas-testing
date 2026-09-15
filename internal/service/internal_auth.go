package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"simas-backend/internal/auth"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
)

var (
	ErrInternalInvalidCredentials = errors.New("invalid credentials")
	ErrPasswordMismatch           = errors.New("password confirmation does not match")
	ErrSamePassword               = errors.New("new password must be different from old password")
	ErrInvalidTempToken           = errors.New("invalid or expired temp token")
	ErrNotInternalAccount         = errors.New("not an internal account")
	ErrForbiddenRole              = errors.New("forbidden role assignment")
)

type InternalAuthService struct {
	userRepo   *repository.UserRepository
	rtRepo     *repository.RefreshTokenRepository
	jwtManager *auth.JWTManager
}

func NewInternalAuthService(
	userRepo *repository.UserRepository,
	rtRepo *repository.RefreshTokenRepository,
	jwtManager *auth.JWTManager,
) *InternalAuthService {
	return &InternalAuthService{
		userRepo:   userRepo,
		rtRepo:     rtRepo,
		jwtManager: jwtManager,
	}
}

// PusatLogin logs in a super_admin user. No force-update required.
func (s *InternalAuthService) PusatLogin(ctx context.Context, email, password string) (*model.SessionResponse, error) {
	user, err := s.userRepo.FindByEmailAndRole(ctx, email, []string{"super_admin"})
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInternalInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if !auth.CheckPassword(strVal(user.PasswordHash), password) {
		return nil, ErrInternalInvalidCredentials
	}

	if user.AccountStatus != "active" {
		return nil, ErrEmailNotVerified
	}

	_ = s.userRepo.SetLastLogin(ctx, user.ID)
	return s.issueSession(user)
}

// InternalLogin logs in admin_sekolah or tenaga_pendidik.
// Returns 202 (temp_token) if is_password_updated == false.
func (s *InternalAuthService) InternalLogin(ctx context.Context, email, password string) (interface{}, error) {
	user, err := s.userRepo.FindByEmailAndRole(ctx, email, []string{"admin_sekolah", "tenaga_pendidik"})
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInternalInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if !auth.CheckPassword(strVal(user.PasswordHash), password) {
		return nil, ErrInternalInvalidCredentials
	}

	if user.AccountStatus != "active" {
		return nil, ErrEmailNotVerified
	}

	// Force password update required
	if !user.IsPasswordUpdated {
		tempToken, _, err := s.jwtManager.GenerateTempToken(user.ID, user.Email)
		if err != nil {
			return nil, fmt.Errorf("generate temp token: %w", err)
		}
		return &model.ForceUpdateResponse{
			AccountState: "require_password_update",
			TempToken:    tempToken,
			ExpiresIn:    int(auth.TempTokenLifetime.Seconds()),
		}, nil
	}

	_ = s.userRepo.SetLastLogin(ctx, user.ID)
	return s.issueSession(user)
}

// ForceUpdatePassword updates password using temp_token.
func (s *InternalAuthService) ForceUpdatePassword(ctx context.Context, tempToken, oldPassword, newPassword, confirmPassword string) (*model.SessionResponse, error) {
	if newPassword != confirmPassword {
		return nil, ErrPasswordMismatch
	}

	if err := auth.ValidatePasswordStrength(newPassword); err != nil {
		return nil, err
	}

	claims, err := s.jwtManager.ValidateTempToken(tempToken)
	if err != nil {
		return nil, ErrInvalidTempToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidTempToken
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	// Only internal roles can force-update
	if user.Role != "admin_sekolah" && user.Role != "tenaga_pendidik" {
		return nil, ErrNotInternalAccount
	}

	if !auth.CheckPassword(strVal(user.PasswordHash), oldPassword) {
		return nil, ErrInternalInvalidCredentials
	}

	if auth.CheckPassword(strVal(user.PasswordHash), newPassword) {
		return nil, ErrSamePassword
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	if err := s.userRepo.UpdatePasswordAndActivate(ctx, user.ID, newHash); err != nil {
		return nil, fmt.Errorf("update password: %w", err)
	}

	// Revoke temp token after use
	_ = s.jwtManager.RevokeByToken(tempToken)

	user.IsPasswordUpdated = true
	user.PasswordHash = &newHash
	return s.issueSession(user)
}

// RegisterSuperAdmin creates a new super_admin account. Only existing super_admin can invoke.
func (s *InternalAuthService) RegisterSuperAdmin(ctx context.Context, email, name, password string, creatorRole string) (*model.SessionResponse, error) {
	if creatorRole != "super_admin" {
		return nil, ErrForbiddenRole
	}

	email = normalizeEmail(email)

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &model.User{
		Email:             email,
		Name:              name,
		Provider:          "email",
		PasswordHash:      &hash,
		Role:              "super_admin",
		AccountStatus:     "active",
		OnboardingStatus:  "completed",
		EmailVerified:     true,
		IsPasswordUpdated: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueSession(user)
}

func (s *InternalAuthService) issueSession(user *model.User) (*model.SessionResponse, error) {
	accessToken, _, err := s.jwtManager.GenerateAccess(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, _, err := s.jwtManager.GenerateRefresh(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	if _, err := s.rtRepo.Store(context.Background(), user.ID, refreshToken, time.Now().Add(auth.RefreshTokenLifetime)); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	return &model.SessionResponse{
		AccountStatus:    user.AccountStatus,
		OnboardingStatus: user.OnboardingStatus,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int(auth.AccessTokenLifetime.Seconds()),
		User:             user,
	}, nil
}
