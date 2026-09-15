package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"simas-backend/internal/model"
)

var ErrPendidikNotFound = errors.New("pendidik not found")
var ErrPendidikExists = errors.New("pendidik already exists")
var ErrJadwalTabrakan = errors.New("jadwal tabrakan detected")

type PendidikRepository struct {
	db *gorm.DB
}

func NewPendidikRepository(db *gorm.DB) *PendidikRepository {
	return &PendidikRepository{db: db}
}

func (r *PendidikRepository) Create(ctx context.Context, pendidik *model.Pendidik) error {
	result := r.db.WithContext(ctx).Create(pendidik)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key") {
			return ErrPendidikExists
		}
		return result.Error
	}
	return nil
}

func (r *PendidikRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Pendidik, error) {
	var pendidik model.Pendidik
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&pendidik).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPendidikNotFound
	}
	return &pendidik, err
}

func (r *PendidikRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Pendidik, error) {
	var pendidik model.Pendidik
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&pendidik).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPendidikNotFound
	}
	return &pendidik, err
}

func (r *PendidikRepository) FindByNip(ctx context.Context, nip string) (*model.Pendidik, error) {
	var pendidik model.Pendidik
	err := r.db.WithContext(ctx).Where("nip = ?", nip).First(&pendidik).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPendidikNotFound
	}
	return &pendidik, err
}

func (r *PendidikRepository) FindByNuptk(ctx context.Context, nuptk string) (*model.Pendidik, error) {
	var pendidik model.Pendidik
	err := r.db.WithContext(ctx).Where("nuptk = ?", nuptk).First(&pendidik).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPendidikNotFound
	}
	return &pendidik, err
}

func (r *PendidikRepository) ListBySekolahID(ctx context.Context, sekolahID uuid.UUID, status string, jabatan string, search string, page int, limit int) ([]model.Pendidik, int64, error) {
	var pendidiks []model.Pendidik
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Pendidik{}).Where("sekolah_id = ?", sekolahID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if jabatan != "" {
		query = query.Where("jabatan = ?", jabatan)
	}
	if search != "" {
		query = query.Where("nama_lengkap ILIKE ?", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Order("nama_lengkap ASC").Offset(offset).Limit(limit).Find(&pendidiks).Error; err != nil {
		return nil, 0, err
	}
	return pendidiks, total, nil
}

func (r *PendidikRepository) Update(ctx context.Context, pendidik *model.Pendidik) error {
	return r.db.WithContext(ctx).Save(pendidik).Error
}

func (r *PendidikRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.Pendidik{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *PendidikRepository) CountBySekolahID(ctx context.Context, sekolahID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Pendidik{}).
		Where("sekolah_id = ? AND status = 'aktif'", sekolahID).
		Count(&count).Error
	return count, err
}

type PendidikJadwalRepository struct {
	db *gorm.DB
}

func NewPendidikJadwalRepository(db *gorm.DB) *PendidikJadwalRepository {
	return &PendidikJadwalRepository{db: db}
}

func (r *PendidikJadwalRepository) Create(ctx context.Context, jadwal *model.PendidikJadwal) error {
	return r.db.WithContext(ctx).Create(jadwal).Error
}

func (r *PendidikJadwalRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.PendidikJadwal, error) {
	var jadwal model.PendidikJadwal
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&jadwal).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPendidikNotFound
	}
	return &jadwal, err
}

func (r *PendidikJadwalRepository) ListByPendidikID(ctx context.Context, pendidikID uuid.UUID, semester string) ([]model.PendidikJadwal, error) {
	var jadwals []model.PendidikJadwal
	query := r.db.WithContext(ctx).Where("pendidik_id = ?", pendidikID)
	if semester != "" {
		query = query.Where("semester = ?", semester)
	}
	err := query.Order("hari, jam_mulai").Find(&jadwals).Error
	return jadwals, err
}

func (r *PendidikJadwalRepository) ListBySekolahIDAndHari(ctx context.Context, sekolahID uuid.UUID, hari string) ([]model.PendidikJadwal, error) {
	var jadwals []model.PendidikJadwal
	err := r.db.WithContext(ctx).
		Where("sekolah_id = ? AND hari = ?", sekolahID, hari).
		Order("jam_mulai").
		Find(&jadwals).Error
	return jadwals, err
}

func (r *PendidikJadwalRepository) CheckTabrakan(ctx context.Context, sekolahID uuid.UUID, hari string, jamMulai string, jamSelesai string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.PendidikJadwal{}).
		Where("sekolah_id = ? AND hari = ? AND status = 'aktif'", sekolahID, hari).
		Where("(jam_mulai < ? AND jam_selesai > ?)", jamSelesai, jamMulai)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PendidikJadwalRepository) Update(ctx context.Context, jadwal *model.PendidikJadwal) error {
	return r.db.WithContext(ctx).Save(jadwal).Error
}

func (r *PendidikJadwalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.PendidikJadwal{}, "id = ?", id).Error
}
