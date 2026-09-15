package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"simas-backend/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user already exists")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key") {
			return ErrUserExists
		}
		return result.Error
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("email = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("google_id = ?", googleID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) FindByEmailAndRole(ctx context.Context, email string, roles []string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND role IN ?", strings.ToLower(strings.TrimSpace(email)), roles).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) UpdatePasswordAndActivate(ctx context.Context, id uuid.UUID, passwordHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"password_hash":       passwordHash,
			"is_password_updated": true,
			"updated_at":          now,
		}).Error
}
func (r *UserRepository) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"email_verified":    true,
			"email_verified_at": now,
			"account_status":    "active",
			"updated_at":        now,
		}).Error
}

func (r *UserRepository) SetLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
}

func (r *UserRepository) FindSuperAdminEmails(ctx context.Context) ([]string, error) {
	var emails []string
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("role = ?", "super_admin").
		Pluck("email", &emails).Error
	return emails, err
}
