package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrTokenExpired = errors.New("token expired")
var ErrTokenInvalid = errors.New("token invalid")
var ErrTokenRevoked = errors.New("token revoked")

const (
	AccessTokenLifetime  = 15 * time.Minute
	RefreshTokenLifetime = 7 * 24 * time.Hour
	TempTokenLifetime    = 15 * time.Minute
)

type JWTManager struct {
	secret []byte
	redis  *redis.Client
}

type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	TokenType string    `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, redisClient *redis.Client) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		redis:  redisClient,
	}
}

// GenerateAccess creates a short-lived access token
func (m *JWTManager) GenerateAccess(userID uuid.UUID, email, role string) (string, string, error) {
	return m.generate(userID, email, role, "access", AccessTokenLifetime)
}

// GenerateRefresh creates a long-lived refresh token, returns (token, jti, error)
func (m *JWTManager) GenerateRefresh(userID uuid.UUID, email, role string) (string, string, error) {
	return m.generate(userID, email, role, "refresh", RefreshTokenLifetime)
}

func (m *JWTManager) generate(userID uuid.UUID, email, role, tokenType string, lifetime time.Duration) (string, string, error) {
	jti := uuid.New().String()
	claims := &Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(lifetime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, jti, nil
}

func (m *JWTManager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	if m.redis != nil {
		blacklisted, _ := m.redis.Get(context.Background(), "blacklist:"+claims.ID).Result()
		if blacklisted == "1" {
			return nil, ErrTokenRevoked
		}
	}
	return claims, nil
}

// GenerateTempToken creates a short-lived temp token for force password update
func (m *JWTManager) GenerateTempToken(userID uuid.UUID, email string) (string, string, error) {
	return m.generate(userID, email, "", "temp", TempTokenLifetime)
}

// ValidateTempToken validates a temp token and returns claims
func (m *JWTManager) ValidateTempToken(tokenString string) (*Claims, error) {
	claims, err := m.Validate(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "temp" {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// Revoke blacklists a token by JTI until its natural expiry
func (m *JWTManager) Revoke(jti string, expiresAt time.Time) error {
	if m.redis == nil {
		return nil
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return m.redis.Set(context.Background(), "blacklist:"+jti, "1", ttl).Err()
}

// RevokeByToken parses and revokes a token string
func (m *JWTManager) RevokeByToken(tokenString string) error {
	claims, err := m.Validate(tokenString)
	if err != nil {
		return nil // token already invalid
	}
	return m.Revoke(claims.ID, claims.ExpiresAt.Time)
}
