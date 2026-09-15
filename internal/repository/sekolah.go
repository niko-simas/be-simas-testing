package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"simas-backend/internal/model"
)

var ErrSekolahNotFound = errors.New("sekolah not found")
var ErrSekolahExists = errors.New("sekolah already exists")

type SekolahRepository struct {
	db *gorm.DB
}

func NewSekolahRepository(db *gorm.DB) *SekolahRepository {
	return &SekolahRepository{db: db}
}

func (r *SekolahRepository) Create(ctx context.Context, sekolah *model.Sekolah) error {
	sekolah.Email = strings.ToLower(strings.TrimSpace(sekolah.Email))
	result := r.db.WithContext(ctx).Create(sekolah)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key") {
			return ErrSekolahExists
		}
		return result.Error
	}
	return nil
}

func (r *SekolahRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Sekolah, error) {
	var sekolah model.Sekolah
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sekolah).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSekolahNotFound
	}
	return &sekolah, err
}

func (r *SekolahRepository) FindByEmail(ctx context.Context, email string) (*model.Sekolah, error) {
	var sekolah model.Sekolah
	err := r.db.WithContext(ctx).
		Where("email = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&sekolah).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSekolahNotFound
	}
	return &sekolah, err
}

func (r *SekolahRepository) FindByNpsn(ctx context.Context, npsn string) (*model.Sekolah, error) {
	var sekolah model.Sekolah
	err := r.db.WithContext(ctx).Where("npsn = ?", npsn).First(&sekolah).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSekolahNotFound
	}
	return &sekolah, err
}

func (r *SekolahRepository) FindByTrackingCode(ctx context.Context, trackingCode string) (*model.Sekolah, error) {
	var sekolah model.Sekolah
	err := r.db.WithContext(ctx).Where("tracking_code = ?", trackingCode).First(&sekolah).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSekolahNotFound
	}
	return &sekolah, err
}

func (r *SekolahRepository) FindByAdminID(ctx context.Context, adminID uuid.UUID) (*model.Sekolah, error) {
	var sekolah model.Sekolah
	err := r.db.WithContext(ctx).Where("admin_sekolah_id = ?", adminID).First(&sekolah).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSekolahNotFound
	}
	return &sekolah, err
}

func (r *SekolahRepository) List(ctx context.Context, status string, search string, page int, limit int) ([]model.Sekolah, int64, error) {
	var sekolahs []model.Sekolah
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Sekolah{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("nama ILIKE ? OR npsn ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&sekolahs).Error; err != nil {
		return nil, 0, err
	}
	return sekolahs, total, nil
}

func (r *SekolahRepository) Update(ctx context.Context, sekolah *model.Sekolah) error {
	return r.db.WithContext(ctx).Save(sekolah).Error
}

func (r *SekolahRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.Sekolah{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *SekolahRepository) UpdateAdminID(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Sekolah{}).
		Where("id = ?", id).
		Update("admin_sekolah_id", adminID).Error
}

type SekolahStateLogRepository struct {
	db *gorm.DB
}

func NewSekolahStateLogRepository(db *gorm.DB) *SekolahStateLogRepository {
	return &SekolahStateLogRepository{db: db}
}

func (r *SekolahStateLogRepository) Create(ctx context.Context, log *model.SekolahStateLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *SekolahStateLogRepository) FindBySekolahID(ctx context.Context, sekolahID uuid.UUID) ([]model.SekolahStateLog, error) {
	var logs []model.SekolahStateLog
	err := r.db.WithContext(ctx).
		Where("sekolah_id = ?", sekolahID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}
