package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Email            string     `gorm:"uniqueIndex;not null" json:"email"`
	Name             string     `gorm:"type:varchar(255)" json:"name,omitempty"`
	GoogleID         *string    `gorm:"type:varchar(255);uniqueIndex" json:"google_id,omitempty"`
	AvatarURL        *string    `gorm:"type:text" json:"avatar_url,omitempty"`
	Provider         string     `gorm:"type:varchar(50);not null;default:'google'" json:"provider"`
	PasswordHash     *string    `gorm:"type:text" json:"-"`
	NoWhatsApp       *string    `gorm:"type:varchar(20)" json:"no_whatsapp,omitempty"`
	Role             string     `gorm:"type:varchar(50);not null;default:'wali_murid'" json:"role"`
	AccountStatus    string     `gorm:"type:varchar(40);not null;default:'pending_verification'" json:"account_status"`
	OnboardingStatus string     `gorm:"type:varchar(40);not null;default:'need_kyc'" json:"onboarding_status"`
	EmailVerified    bool       `gorm:"not null;default:false" json:"email_verified"`
	EmailVerifiedAt  *time.Time `json:"email_verified_at,omitempty"`
	IsPasswordUpdated bool      `gorm:"not null;default:false" json:"is_password_updated"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	CreatedAt        time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

func (User) TableName() string { return "users" }

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type AuthResponse struct {
	User        *User  `json:"user"`
	AccessToken string `json:"access_token"`
	IsNewUser   bool   `json:"is_new_user"`
}

type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}
