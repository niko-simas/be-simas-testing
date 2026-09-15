package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"simas-backend/internal/auth"
	"simas-backend/internal/infrastructure/mailer"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
)

var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidOTP         = errors.New("invalid or expired otp")
	ErrResendCooldown     = errors.New("please wait before resending otp")
	ErrPendingOAuth       = errors.New("pending oauth session not found or expired")
	ErrInvalidWhatsApp    = errors.New("invalid whatsapp number")
	ErrGoogleTokenInvalid = errors.New("invalid google token")
)

type MobileAuthService struct {
	userRepo    *repository.UserRepository
	rtRepo      *repository.RefreshTokenRepository
	otpService  *OTPService
	jwtManager  *auth.JWTManager
	googleOAuth *auth.GoogleOAuth
	mailer      *mailer.SMTPMailer
	redis       *redis.Client
}

func NewMobileAuthService(
	userRepo *repository.UserRepository,
	rtRepo *repository.RefreshTokenRepository,
	otpService *OTPService,
	jwtManager *auth.JWTManager,
	googleOAuth *auth.GoogleOAuth,
	smtpMailer *mailer.SMTPMailer,
	redisClient *redis.Client,
) *MobileAuthService {
	return &MobileAuthService{
		userRepo:    userRepo,
		rtRepo:      rtRepo,
		otpService:  otpService,
		jwtManager:  jwtManager,
		googleOAuth: googleOAuth,
		mailer:      smtpMailer,
		redis:       redisClient,
	}
}

// ---- Normalizers ----

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var waRegex = regexp.MustCompile(`^(\+62|62|0)8[0-9]{7,13}$`)

func normalizeWhatsApp(wa string) (string, error) {
	wa = strings.TrimSpace(strings.ReplaceAll(wa, " ", ""))
	wa = strings.ReplaceAll(wa, "-", "")
	if !waRegex.MatchString(wa) {
		return "", ErrInvalidWhatsApp
	}
	if strings.HasPrefix(wa, "+62") {
		wa = wa[1:]
	} else if strings.HasPrefix(wa, "0") {
		wa = "62" + wa[1:]
	}
	return wa, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---- Register ----

func (s *MobileAuthService) Register(ctx context.Context, req *model.RegisterRequest) error {
	email := normalizeEmail(req.Email)

	wa, err := normalizeWhatsApp(req.NoWhatsApp)
	if err != nil {
		return err
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return fmt.Errorf("check existing user: %w", err)
	}

	if existing != nil {
		if existing.AccountStatus == "active" {
			return ErrEmailAlreadyExists
		}
		// Resend OTP for pending account
		otp, err := s.otpService.Generate(ctx, email)
		if err != nil {
			if errors.Is(err, ErrOTPResendWait) {
				return ErrResendCooldown
			}
			return err
		}
		s.sendOTPEmail(email, existing.Name, otp)
		return nil
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Email:            email,
		Name:             strings.TrimSpace(req.NamaLengkap),
		Provider:         "email",
		PasswordHash:     strPtr(hash),
		NoWhatsApp:       strPtr(wa),
		Role:             "wali_murid",
		AccountStatus:    "pending_verification",
		OnboardingStatus: "need_kyc",
		EmailVerified:    false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}

	otp, err := s.otpService.Generate(ctx, email)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	s.sendOTPEmail(email, user.Name, otp)
	return nil
}

// ---- Verify OTP ----

func (s *MobileAuthService) VerifyOTP(ctx context.Context, req *model.VerifyOTPRequest) (*model.SessionResponse, error) {
	email := normalizeEmail(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidOTP // generic to avoid email enumeration
		}
		return nil, err
	}

	if user.AccountStatus == "active" {
		return nil, errors.New("account already verified")
	}

	if err := s.otpService.Verify(ctx, email, req.OTPCode); err != nil {
		if errors.Is(err, ErrOTPResendWait) {
			return nil, ErrResendCooldown
		}
		return nil, ErrInvalidOTP
	}

	if err := s.userRepo.SetEmailVerified(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("set email verified: %w", err)
	}

	user.AccountStatus = "active"
	return s.IssueSession(user)
}

// ---- Resend OTP ----

func (s *MobileAuthService) ResendOTP(ctx context.Context, req *model.ResendOTPRequest) error {
	email := normalizeEmail(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Return nil silently to avoid email enumeration
		return nil
	}

	if user.AccountStatus == "active" {
		return nil
	}

	otp, err := s.otpService.Generate(ctx, email)
	if err != nil {
		if errors.Is(err, ErrOTPResendWait) {
			return ErrResendCooldown
		}
		return err
	}

	s.sendOTPEmail(email, user.Name, otp)
	return nil
}

// ---- Login ----

func (s *MobileAuthService) Login(ctx context.Context, req *model.MobileLoginRequest) (*model.SessionResponse, error) {
	email := normalizeEmail(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.Provider != "email" {
		return nil, ErrInvalidCredentials
	}

	if !auth.CheckPassword(strVal(user.PasswordHash), req.Password) {
		return nil, ErrInvalidCredentials
	}

	if user.AccountStatus != "active" {
		return nil, ErrEmailNotVerified
	}

	s.userRepo.SetLastLogin(ctx, user.ID)
	return s.IssueSession(user)
}

// ---- Google OAuth Mobile ----

func (s *MobileAuthService) OAuth(ctx context.Context, googleToken string) (interface{}, error) {
	googleUser, err := s.googleOAuth.ValidateIDToken(googleToken)
	if err != nil {
		return nil, ErrGoogleTokenInvalid
	}
	if !googleUser.VerifiedEmail {
		return nil, errors.New("google email not verified")
	}

	ctx = context.Background()

	// Find by Google ID first
	user, err := s.userRepo.FindByGoogleID(ctx, googleUser.ID)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("find by google id: %w", err)
	}

	// Fallback to email
	if user == nil {
		user, err = s.userRepo.FindByEmail(ctx, googleUser.Email)
		if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
			return nil, fmt.Errorf("find by email: %w", err)
		}
		if user != nil {
			// Link Google account
			user.GoogleID = strPtr(googleUser.ID)
			if user.AvatarURL == nil {
				user.AvatarURL = strPtr(googleUser.Picture)
			}
			user.EmailVerified = true
			if user.AccountStatus == "pending_verification" {
				user.AccountStatus = "active"
				now := time.Now()
				user.EmailVerifiedAt = &now
			}
			if err := s.userRepo.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("link google account: %w", err)
			}
		}
	}

	// Existing user with WhatsApp → direct session
	if user != nil && strVal(user.NoWhatsApp) != "" && user.AccountStatus == "active" {
		s.userRepo.SetLastLogin(ctx, user.ID)
		return s.IssueSession(user)
	}

	// Need WhatsApp completion
	tempID, err := auth.GenerateOpaqueID()
	if err != nil {
		return nil, fmt.Errorf("generate temp id: %w", err)
	}

	pending := model.PendingOAuth{
		Provider:      "google",
		GoogleID:      googleUser.ID,
		Email:         googleUser.Email,
		Name:          googleUser.Name,
		AvatarURL:     googleUser.Picture,
		EmailVerified: true,
	}
	if user != nil {
		pending.ExistingUserID = user.ID.String()
	}

	if err := SavePendingOAuth(ctx, s.redis, tempID, pending); err != nil {
		return nil, fmt.Errorf("save pending oauth: %w", err)
	}

	return &model.OAuthPendingResponse{
		AccountState: "require_whatsapp",
		TempUserID:   tempID,
		ExpiresIn:    600,
	}, nil
}

// ---- Complete OAuth (add WhatsApp) ----

func (s *MobileAuthService) CompleteOAuth(ctx context.Context, req *model.CompleteOAuthRequest) (*model.SessionResponse, error) {
	wa, err := normalizeWhatsApp(req.NoWhatsApp)
	if err != nil {
		return nil, err
	}

	var pending model.PendingOAuth
	if err := GetPendingOAuth(ctx, s.redis, req.TempUserID, &pending); err != nil {
		return nil, ErrPendingOAuth
	}

	var user *model.User
	now := time.Now()

	if pending.ExistingUserID != "" {
		// Update existing user with WhatsApp
		userID, err := uuid.Parse(pending.ExistingUserID)
		if err != nil {
			return nil, ErrPendingOAuth
		}
		user, err = s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return nil, ErrPendingOAuth
		}
		user.NoWhatsApp = strPtr(wa)
		user.AccountStatus = "active"
		user.EmailVerified = true
		user.EmailVerifiedAt = &now
		if user.GoogleID == nil {
			user.GoogleID = strPtr(pending.GoogleID)
		}
		if user.AvatarURL == nil {
			user.AvatarURL = strPtr(pending.AvatarURL)
		}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	} else {
		// Create new user
		user = &model.User{
			Email:            normalizeEmail(pending.Email),
			Name:             pending.Name,
			GoogleID:         strPtr(pending.GoogleID),
			AvatarURL:        strPtr(pending.AvatarURL),
			Provider:         "google",
			NoWhatsApp:       strPtr(wa),
			Role:             "wali_murid",
			AccountStatus:    "active",
			OnboardingStatus: "need_kyc",
			EmailVerified:    true,
			EmailVerifiedAt:  &now,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			if errors.Is(err, repository.ErrUserExists) {
				return nil, ErrEmailAlreadyExists
			}
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	return s.IssueSession(user)
}

// ---- Refresh ----

func (s *MobileAuthService) Refresh(ctx context.Context, refreshToken string) (*model.SessionResponse, error) {
	userID, recordID, err := s.rtRepo.Validate(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate new tokens
	accessToken, _, err := s.jwtManager.GenerateAccess(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}
	newRefreshToken, _, err := s.jwtManager.GenerateRefresh(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	// Store new refresh token and revoke old
	newRecordID, err := s.rtRepo.Store(ctx, user.ID, newRefreshToken, time.Now().Add(auth.RefreshTokenLifetime))
	if err != nil {
		return nil, err
	}
	s.rtRepo.Revoke(ctx, recordID, newRecordID)

	return &model.SessionResponse{
		AccountStatus:    user.AccountStatus,
		OnboardingStatus: user.OnboardingStatus,
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		ExpiresIn:        int(auth.AccessTokenLifetime.Seconds()),
	}, nil
}

// ValidateRefreshToken exposes refresh token validation
func (s *MobileAuthService) ValidateRefreshToken(ctx context.Context, token string) (uuid.UUID, string, error) {
	return s.rtRepo.Validate(ctx, token)
}

// RevokeRefreshToken exposes refresh token revocation
func (s *MobileAuthService) RevokeRefreshToken(ctx context.Context, recordID string, replacedByID string) error {
	return s.rtRepo.Revoke(ctx, recordID, replacedByID)
}

func (s *MobileAuthService) GetStatus(user *model.User) *model.StatusVerifikasiResponse {
	return &model.StatusVerifikasiResponse{
		Status:        user.OnboardingStatus,
		AccountStatus: user.AccountStatus,
	}
}

// ---- Helpers ----

func (s *MobileAuthService) IssueSession(user *model.User) (*model.SessionResponse, error) {
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
	}, nil
}

func (s *MobileAuthService) sendOTPEmail(to, name, otp string) {
	subject := "Kode Verifikasi SIMAS"
	body := mailer.BuildOTPEmail(name, otp)
	s.mailer.SendAsync(to, subject, body)
}
