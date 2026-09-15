package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash    string     `gorm:"type:text;uniqueIndex;not null" json:"-"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	ReplacedByID *uuid.UUID `gorm:"type:uuid" json:"replaced_by_id,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// ---- Request DTOs ----

type RegisterRequest struct {
	NamaLengkap string `json:"nama_lengkap" binding:"required,min=3,max=255"`
	Email       string `json:"email" binding:"required,email"`
	NoWhatsApp  string `json:"no_whatsapp" binding:"required,min=9,max=15"`
	Password    string `json:"password" binding:"required,min=8"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"otp_code" binding:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type MobileLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type MobileOAuthRequest struct {
	GoogleToken string `json:"google_token" binding:"required"`
}

type CompleteOAuthRequest struct {
	TempUserID string `json:"temp_user_id" binding:"required"`
	NoWhatsApp string `json:"no_whatsapp" binding:"required,min=9,max=15"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ---- Response DTOs ----

type RegisterResponse struct {
	Message   string `json:"message"`
	Email     string `json:"email"`
	ExpiresIn int    `json:"expires_in"`
}

type ResendResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"`
}

type SessionResponse struct {
	AccountStatus    string `json:"account_status"`
	OnboardingStatus string `json:"onboarding_status"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	User             *User  `json:"user,omitempty"`
}

type OAuthPendingResponse struct {
	AccountState string `json:"account_state"`
	TempUserID   string `json:"temp_user_id"`
	ExpiresIn    int    `json:"expires_in"`
	Name         string `json:"name,omitempty"`
	Email        string `json:"email,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
}

type StatusVerifikasiResponse struct {
	Status        string `json:"status"`
	AccountStatus string `json:"account_status"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ForceUpdateResponse struct {
	AccountState string `json:"account_state"`
	TempToken    string `json:"temp_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// PendingOAuth session stored in Redis
type PendingOAuth struct {
	Provider       string `json:"provider"`
	GoogleID       string `json:"google_id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	AvatarURL      string `json:"avatar_url"`
	EmailVerified  bool   `json:"email_verified"`
	ExistingUserID string `json:"existing_user_id,omitempty"`
}
