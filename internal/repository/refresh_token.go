package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"simas-backend/internal/model"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")
var ErrRefreshTokenRevoked = errors.New("refresh token revoked")
var ErrRefreshTokenExpired = errors.New("refresh token expired")

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (r *RefreshTokenRepository) Store(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (string, error) {
	rt := &model.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(token),
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(rt).Error; err != nil {
		return "", err
	}
	return rt.ID.String(), nil
}

func (r *RefreshTokenRepository) Validate(ctx context.Context, token string) (uuid.UUID, string, error) {
	var rt model.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", hashToken(token)).
		First(&rt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, "", ErrRefreshTokenNotFound
	}
	if err != nil {
		return uuid.Nil, "", err
	}
	if rt.RevokedAt != nil {
		return uuid.Nil, "", ErrRefreshTokenRevoked
	}
	if time.Now().After(rt.ExpiresAt) {
		return uuid.Nil, "", ErrRefreshTokenExpired
	}
	return rt.UserID, rt.ID.String(), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, recordID string, replacedByID string) error {
	now := time.Now()
	updates := map[string]interface{}{"revoked_at": now}
	if replacedByID != "" {
		if uid, err := uuid.Parse(replacedByID); err == nil {
			updates["replaced_by_id"] = uid
		}
	}
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("id = ?", recordID).
		Updates(updates).Error
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}
