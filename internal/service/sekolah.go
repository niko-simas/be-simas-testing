package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"simas-backend/internal/auth"
	"simas-backend/internal/infrastructure/mailer"
	"simas-backend/internal/infrastructure/storage"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
)

var (
	ErrSekolahExists              = errors.New("sekolah already exists")
	ErrSekolahNotFound            = errors.New("sekolah not found")
	ErrInvalidJenjang             = errors.New("invalid jenjang")
	ErrInvalidStatus              = errors.New("invalid status transition")
	ErrSekolahNotOwned            = errors.New("sekolah not owned by this user")
	ErrIncompleteLegalitas        = errors.New("incomplete legalitas documents")
	ErrFileTooLarge               = errors.New("file too large, max 3MB")
	ErrInvalidFileFormat          = errors.New("invalid file format, must be PDF")
	ErrPengajuanUlangNotAllowed   = errors.New("pengajuan ulang only allowed for ditolak status")
	ErrInvalidTrackingCode        = errors.New("invalid tracking code")
)

const maxFileSize = 3 * 1024 * 1024 // 3MB

var validJenjang = map[string]bool{
	"sd_mi": true, "smp_mts": true, "sma_smk_ma_mak": true,
}

var validStatusTransitions = map[string][]string{
	"pengajuan":           {"menunggu_verifikasi", "ditolak"},
	"menunggu_verifikasi":  {"aktif", "ditolak"},
	"aktif":               {"nonaktif"},
	"nonaktif":            {"aktif"},
	"ditolak":             {"pengajuan"},
}

var statusLabels = map[string]string{
	"pengajuan":           "Pengajuan Diterima",
	"menunggu_verifikasi": "Menunggu Verifikasi Admin",
	"aktif":               "Aktif",
	"nonaktif":            "Nonaktif",
	"ditolak":             "Ditolak",
}

type SekolahService struct {
	sekolahRepo    *repository.SekolahRepository
	sekolahLogRepo *repository.SekolahStateLogRepository
	userRepo       *repository.UserRepository
	jwtManager     *auth.JWTManager
	mailer         *mailer.SMTPMailer
	storage        storage.Storage
}

func NewSekolahService(
	sekolahRepo *repository.SekolahRepository,
	sekolahLogRepo *repository.SekolahStateLogRepository,
	userRepo *repository.UserRepository,
	jwtManager *auth.JWTManager,
	mailer *mailer.SMTPMailer,
	storage storage.Storage,
) *SekolahService {
	return &SekolahService{
		sekolahRepo:    sekolahRepo,
		sekolahLogRepo: sekolahLogRepo,
		userRepo:       userRepo,
		jwtManager:     jwtManager,
		mailer:         mailer,
		storage:        storage,
	}
}

func generateTrackingCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	return "SM" + string(code)
}

func generateRandomPassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	password := make([]byte, 12)
	for i := range password {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[n.Int64()]
	}
	return string(password), nil
}

// ---- Step 1: Submit Data Sekolah (JSON) ----

func (s *SekolahService) PengajuanBaru(ctx context.Context, req *model.SekolahRequest) (*model.Sekolah, error) {
	if !validJenjang[req.Jenjang] {
		return nil, ErrInvalidJenjang
	}
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return nil, errors.New("latitude must be between -90 and 90")
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return nil, errors.New("longitude must be between -180 and 180")
	}

	_, err := s.sekolahRepo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrSekolahExists
	}
	if !errors.Is(err, repository.ErrSekolahNotFound) {
		return nil, fmt.Errorf("check email: %w", err)
	}

	if req.Npsn != nil && *req.Npsn != "" {
		_, err = s.sekolahRepo.FindByNpsn(ctx, *req.Npsn)
		if err == nil {
			return nil, ErrSekolahExists
		}
		if !errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, fmt.Errorf("check npsn: %w", err)
		}
	}

	var tanggalBerdiri *time.Time
	if req.TanggalBerdiri != nil && *req.TanggalBerdiri != "" {
		t, err := time.Parse("2006-01-02", *req.TanggalBerdiri)
		if err != nil {
			return nil, errors.New("invalid tanggal_berdiri format, expected YYYY-MM-DD")
		}
		tanggalBerdiri = &t
	}

	sekolah := &model.Sekolah{
		Nama:              req.Nama,
		Npsn:              req.Npsn,
		Jenjang:           req.Jenjang,
		Status:            "pengajuan",
		Alamat:            req.Alamat,
		ProvinsiID:        req.ProvinsiID,
		KabupatenID:       req.KabupatenID,
		KecamatanID:       req.KecamatanID,
		DesaID:            req.DesaID,
		KodePos:           req.KodePos,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		Email:             strings.ToLower(strings.TrimSpace(req.Email)),
		Telepon:           req.Telepon,
		Website:           req.Website,
		Yayasan:           req.Yayasan,
		KepalaSekolahNama: req.KepalaSekolahNama,
		KepalaSekolahNip:  req.KepalaSekolahNip,
		TanggalBerdiri:    tanggalBerdiri,
		SkPendirianNomor:  req.SkPendirianNomor,
		TrackingCode:      generateTrackingCode(),
	}

	if err := s.sekolahRepo.Create(ctx, sekolah); err != nil {
		return nil, fmt.Errorf("create sekolah: %w", err)
	}

	return sekolah, nil
}

// ---- Step 2: Upload Dokumen (Multipart Form-Data) ----

func (s *SekolahService) UploadDokumen(ctx context.Context, sekolahID uuid.UUID, files map[string][]byte) (*model.Sekolah, error) {
	sekolah, err := s.sekolahRepo.FindByID(ctx, sekolahID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	if sekolah.Status != "pengajuan" {
		return nil, errors.New("dokumen hanya bisa diupload saat status pengajuan")
	}

	if err := s.uploadFilesAndUpdate(ctx, sekolah, files); err != nil {
		return nil, err
	}

	return sekolah, nil
}

// ---- Step 1+2: Pengajuan Baru + Upload Dokumen sekaligus ----

func (s *SekolahService) PengajuanBaruLengkap(ctx context.Context, req *model.SekolahRequest, files map[string][]byte) (*model.Sekolah, error) {
	if !validJenjang[req.Jenjang] {
		return nil, ErrInvalidJenjang
	}
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return nil, errors.New("latitude must be between -90 and 90")
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return nil, errors.New("longitude must be between -180 and 180")
	}

	_, err := s.sekolahRepo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrSekolahExists
	}
	if !errors.Is(err, repository.ErrSekolahNotFound) {
		return nil, fmt.Errorf("check email: %w", err)
	}

	if req.Npsn != nil && *req.Npsn != "" {
		_, err = s.sekolahRepo.FindByNpsn(ctx, *req.Npsn)
		if err == nil {
			return nil, ErrSekolahExists
		}
		if !errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, fmt.Errorf("check npsn: %w", err)
		}
	}

	var tanggalBerdiri *time.Time
	if req.TanggalBerdiri != nil && *req.TanggalBerdiri != "" {
		t, err := time.Parse("2006-01-02", *req.TanggalBerdiri)
		if err != nil {
			return nil, errors.New("invalid tanggal_berdiri format, expected YYYY-MM-DD")
		}
		tanggalBerdiri = &t
	}

	sekolah := &model.Sekolah{
		Nama:              req.Nama,
		Npsn:              req.Npsn,
		Jenjang:           req.Jenjang,
		Status:            "pengajuan",
		Alamat:            req.Alamat,
		ProvinsiID:        req.ProvinsiID,
		KabupatenID:       req.KabupatenID,
		KecamatanID:       req.KecamatanID,
		DesaID:            req.DesaID,
		KodePos:           req.KodePos,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		Email:             strings.ToLower(strings.TrimSpace(req.Email)),
		Telepon:           req.Telepon,
		Website:           req.Website,
		Yayasan:           req.Yayasan,
		KepalaSekolahNama: req.KepalaSekolahNama,
		KepalaSekolahNip:  req.KepalaSekolahNip,
		TanggalBerdiri:    tanggalBerdiri,
		SkPendirianNomor:  req.SkPendirianNomor,
		TrackingCode:      generateTrackingCode(),
	}

	if err := s.sekolahRepo.Create(ctx, sekolah); err != nil {
		return nil, fmt.Errorf("create sekolah: %w", err)
	}

	if err := s.uploadFilesAndUpdate(ctx, sekolah, files); err != nil {
		return nil, err
	}

	return sekolah, nil
}

func (s *SekolahService) uploadFilesAndUpdate(ctx context.Context, sekolah *model.Sekolah, files map[string][]byte) error {
	requiredFields := []string{"akta_pendirian", "nib", "sk_pendirian", "siop"}
	for _, field := range requiredFields {
		data, ok := files[field]
		if !ok || len(data) == 0 {
			return ErrIncompleteLegalitas
		}
		if len(data) > maxFileSize {
			return ErrFileTooLarge
		}
		if len(data) < 4 || string(data[:4]) != "%PDF" {
			return ErrInvalidFileFormat
		}
	}

	now := time.Now()
	sekolahID := sekolah.ID
	keyMap := map[string]**string{
		"akta_pendirian": &sekolah.AktaPendirianURL,
		"nib":            &sekolah.NibURL,
		"sk_pendirian":   &sekolah.SkPendirianURL,
		"siop":           &sekolah.SiopURL,
	}

	for field, data := range files {
		targetPtr, ok := keyMap[field]
		if !ok {
			continue
		}
		key := storage.BuildSekolahLegalitasKey(sekolahID.String(), field+".pdf")
		url, err := s.storage.Upload(ctx, key, data, "application/pdf")
		if err != nil {
			return fmt.Errorf("upload %s: %w", field, err)
		}
		*targetPtr = &url
	}

	oldStatus := sekolah.Status
	sekolah.Status = "menunggu_verifikasi"
	sekolah.UpdatedAt = now

	if err := s.sekolahRepo.Update(ctx, sekolah); err != nil {
		return fmt.Errorf("update sekolah: %w", err)
	}

	log := &model.SekolahStateLog{
		SekolahID:  sekolahID,
		StatusLama: &oldStatus,
		StatusBaru: "menunggu_verifikasi",
		Catatan:    &[]string{"Dokumen legalitas telah diupload"}[0],
		CreatedAt:  now,
	}
	_ = s.sekolahLogRepo.Create(ctx, log)

	emails, err := s.userRepo.FindSuperAdminEmails(ctx)
	if err == nil && len(emails) > 0 {
		subject := fmt.Sprintf("[SIMAS] Pengajuan Sekolah Menunggu Verifikasi: %s", sekolah.Nama)
		body := mailer.BuildSekolahPengajuanNotifEmail(sekolah.Nama, sekolah.Email, sekolah.Jenjang)
		for _, email := range emails {
			s.mailer.SendAsync(email, subject, body)
		}
	}

	return nil
}

// ---- Verifikasi oleh Super Admin ----

func (s *SekolahService) Verifikasi(ctx context.Context, sekolahID uuid.UUID, superAdminID uuid.UUID, statusBaru string, catatan *string) (*model.Sekolah, error) {
	sekolah, err := s.sekolahRepo.FindByID(ctx, sekolahID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	validTransitions, ok := validStatusTransitions[sekolah.Status]
	if !ok {
		return nil, ErrInvalidStatus
	}
	found := false
	for _, t := range validTransitions {
		if t == statusBaru {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrInvalidStatus
	}

	statusLama := sekolah.Status
	sekolah.Status = statusBaru

	if statusBaru == "aktif" && statusLama == "menunggu_verifikasi" {
		password, err := generateRandomPassword()
		if err != nil {
			return nil, fmt.Errorf("generate password: %w", err)
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}

		now := time.Now()
		adminUser := &model.User{
			Email:             sekolah.Email,
			Name:              sekolah.Nama,
			Provider:          "email",
			PasswordHash:      &hash,
			Role:              "admin_sekolah",
			AccountStatus:     "active",
			OnboardingStatus:  "completed",
			EmailVerified:     true,
			IsPasswordUpdated: false,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if err := s.userRepo.Create(ctx, adminUser); err != nil {
			if errors.Is(err, repository.ErrUserExists) {
				return nil, ErrEmailAlreadyExists
			}
			return nil, fmt.Errorf("create admin_sekolah: %w", err)
		}

		sekolah.AdminSekolahID = &adminUser.ID

		loginURL := "https://simas.id/login"
		credBody := mailer.BuildKredensialEmail(sekolah.Nama, "admin_sekolah", sekolah.Email, password, loginURL)
		s.mailer.SendAsync(sekolah.Email, "[SIMAS] Kredensial Akun Administrator Sekolah", credBody)
	}

	if err := s.sekolahRepo.Update(ctx, sekolah); err != nil {
		return nil, fmt.Errorf("update sekolah: %w", err)
	}

	log := &model.SekolahStateLog{
		SekolahID:    sekolahID,
		StatusLama:   &statusLama,
		StatusBaru:   statusBaru,
		Catatan:      catatan,
		DiprosesOleh: &superAdminID,
		CreatedAt:    time.Now(),
	}
	if err := s.sekolahLogRepo.Create(ctx, log); err != nil {
		fmt.Printf("failed to create state log: %v\n", err)
	}

	switch statusBaru {
	case "aktif":
		body := mailer.BuildSekolahDiterimaEmail(sekolah.Nama)
		s.mailer.SendAsync(sekolah.Email, "[SIMAS] Pengajuan Sekolah Diterima", body)
	case "ditolak":
		alasan := "Tidak ada catatan"
		if catatan != nil && *catatan != "" {
			alasan = *catatan
		}
		body := mailer.BuildSekolahDitolakEmail(sekolah.Nama, alasan)
		s.mailer.SendAsync(sekolah.Email, "[SIMAS] Pengajuan Sekolah Ditolak", body)
	}

	return sekolah, nil
}

// ---- Progress Tracking (Public, tanpa auth) ----

func (s *SekolahService) GetProgress(ctx context.Context, trackingCode string) (*model.SekolahProgressResponse, error) {
	sekolah, err := s.sekolahRepo.FindByTrackingCode(ctx, trackingCode)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrInvalidTrackingCode
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	logs, _ := s.sekolahLogRepo.FindBySekolahID(ctx, sekolah.ID)

	var logSummaries []model.SekolahStateLogSummary
	for _, log := range logs {
		logSummaries = append(logSummaries, model.SekolahStateLogSummary{
			StatusBaru:  log.StatusBaru,
			StatusLabel: statusLabels[log.StatusBaru],
			Catatan:     log.Catatan,
			CreatedAt:   log.CreatedAt,
		})
	}

	return &model.SekolahProgressResponse{
		TrackingCode: sekolah.TrackingCode,
		Nama:         sekolah.Nama,
		Status:       sekolah.Status,
		StatusLabel:  statusLabels[sekolah.Status],
		Email:        sekolah.Email,
		CreatedAt:    sekolah.CreatedAt,
		UpdatedAt:    sekolah.UpdatedAt,
		StateLogs:    logSummaries,
	}, nil
}

// ---- Standard CRUD ----

func (s *SekolahService) ListSekolah(ctx context.Context, status string, search string, page int, limit int) ([]model.Sekolah, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.sekolahRepo.List(ctx, status, search, page, limit)
}

func (s *SekolahService) GetSekolah(ctx context.Context, id uuid.UUID) (*model.Sekolah, []model.SekolahStateLog, error) {
	sekolah, err := s.sekolahRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, nil, ErrSekolahNotFound
		}
		return nil, nil, fmt.Errorf("find sekolah: %w", err)
	}

	logs, err := s.sekolahLogRepo.FindBySekolahID(ctx, id)
	if err != nil {
		logs = []model.SekolahStateLog{}
	}

	return sekolah, logs, nil
}

func (s *SekolahService) GetMySekolah(ctx context.Context, adminID uuid.UUID) (*model.Sekolah, error) {
	sekolah, err := s.sekolahRepo.FindByAdminID(ctx, adminID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}
	return sekolah, nil
}

func (s *SekolahService) UpdateMySekolah(ctx context.Context, adminID uuid.UUID, req *model.SekolahUpdateRequest) (*model.Sekolah, error) {
	sekolah, err := s.sekolahRepo.FindByAdminID(ctx, adminID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	if req.Nama != nil {
		sekolah.Nama = *req.Nama
	}
	if req.Alamat != nil {
		sekolah.Alamat = req.Alamat
	}
	if req.KodePos != nil {
		sekolah.KodePos = req.KodePos
	}
	if req.Latitude != nil {
		sekolah.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		sekolah.Longitude = req.Longitude
	}
	if req.Telepon != nil {
		sekolah.Telepon = req.Telepon
	}
	if req.Website != nil {
		sekolah.Website = req.Website
	}
	if req.LogoURL != nil {
		sekolah.LogoURL = req.LogoURL
	}

	if err := s.sekolahRepo.Update(ctx, sekolah); err != nil {
		return nil, fmt.Errorf("update sekolah: %w", err)
	}
	return sekolah, nil
}

func (s *SekolahService) UpdateSekolah(ctx context.Context, sekolahID uuid.UUID, req *model.SekolahUpdateRequest) (*model.Sekolah, error) {
	sekolah, err := s.sekolahRepo.FindByID(ctx, sekolahID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	if req.Nama != nil {
		sekolah.Nama = *req.Nama
	}
	if req.Npsn != nil {
		sekolah.Npsn = req.Npsn
	}
	if req.Jenjang != nil {
		if !validJenjang[*req.Jenjang] {
			return nil, ErrInvalidJenjang
		}
		sekolah.Jenjang = *req.Jenjang
	}
	if req.Alamat != nil {
		sekolah.Alamat = req.Alamat
	}
	if req.KodePos != nil {
		sekolah.KodePos = req.KodePos
	}
	if req.Latitude != nil {
		sekolah.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		sekolah.Longitude = req.Longitude
	}
	if req.Telepon != nil {
		sekolah.Telepon = req.Telepon
	}
	if req.Website != nil {
		sekolah.Website = req.Website
	}
	if req.LogoURL != nil {
		sekolah.LogoURL = req.LogoURL
	}
	if req.Yayasan != nil {
		sekolah.Yayasan = req.Yayasan
	}
	if req.KepalaSekolahNama != nil {
		sekolah.KepalaSekolahNama = req.KepalaSekolahNama
	}
	if req.KepalaSekolahNip != nil {
		sekolah.KepalaSekolahNip = req.KepalaSekolahNip
	}
	if req.SkPendirianNomor != nil {
		sekolah.SkPendirianNomor = req.SkPendirianNomor
	}

	if err := s.sekolahRepo.Update(ctx, sekolah); err != nil {
		return nil, fmt.Errorf("update sekolah: %w", err)
	}
	return sekolah, nil
}

func (s *SekolahService) PengajuanUlang(ctx context.Context, trackingCode string, req *model.SekolahRequest) (*model.Sekolah, error) {
	sekolah, err := s.sekolahRepo.FindByTrackingCode(ctx, trackingCode)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	if sekolah.Status != "ditolak" {
		return nil, ErrPengajuanUlangNotAllowed
	}

	if !validJenjang[req.Jenjang] {
		return nil, ErrInvalidJenjang
	}

	var tanggalBerdiri *time.Time
	if req.TanggalBerdiri != nil && *req.TanggalBerdiri != "" {
		t, err := time.Parse("2006-01-02", *req.TanggalBerdiri)
		if err != nil {
			return nil, errors.New("invalid tanggal_berdiri format, expected YYYY-MM-DD")
		}
		tanggalBerdiri = &t
	}

	oldStatus := sekolah.Status
	sekolah.Nama = req.Nama
	sekolah.Npsn = req.Npsn
	sekolah.Jenjang = req.Jenjang
	sekolah.Status = "pengajuan"
	sekolah.Alamat = req.Alamat
	sekolah.ProvinsiID = req.ProvinsiID
	sekolah.KabupatenID = req.KabupatenID
	sekolah.KecamatanID = req.KecamatanID
	sekolah.DesaID = req.DesaID
	sekolah.KodePos = req.KodePos
	sekolah.Latitude = req.Latitude
	sekolah.Longitude = req.Longitude
	sekolah.Email = strings.ToLower(strings.TrimSpace(req.Email))
	sekolah.Telepon = req.Telepon
	sekolah.Website = req.Website
	sekolah.Yayasan = req.Yayasan
	sekolah.KepalaSekolahNama = req.KepalaSekolahNama
	sekolah.KepalaSekolahNip = req.KepalaSekolahNip
	sekolah.TanggalBerdiri = tanggalBerdiri
	sekolah.SkPendirianNomor = req.SkPendirianNomor
	sekolah.AktaPendirianURL = nil
	sekolah.NibURL = nil
	sekolah.SkPendirianURL = nil
	sekolah.SiopURL = nil

	if err := s.sekolahRepo.Update(ctx, sekolah); err != nil {
		return nil, fmt.Errorf("update sekolah: %w", err)
	}

	log := &model.SekolahStateLog{
		SekolahID:  sekolah.ID,
		StatusLama: &oldStatus,
		StatusBaru: "pengajuan",
		Catatan:    &[]string{"Pengajuan ulang setelah penolakan"}[0],
		CreatedAt:  time.Now(),
	}
	_ = s.sekolahLogRepo.Create(ctx, log)

	return sekolah, nil
}

func (s *SekolahService) GetDokumenLegalitas(ctx context.Context, sekolahID uuid.UUID) (map[string]string, error) {
	sekolah, err := s.sekolahRepo.FindByID(ctx, sekolahID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, ErrSekolahNotFound
		}
		return nil, fmt.Errorf("find sekolah: %w", err)
	}

	result := make(map[string]string)
	getExpiry := 15 * time.Minute

	if sekolah.AktaPendirianURL != nil && *sekolah.AktaPendirianURL != "" {
		url, err := s.storage.GeneratePresignedGetURL(ctx, *sekolah.AktaPendirianURL, getExpiry)
		if err == nil {
			result["akta_pendirian_url"] = url
		}
	}
	if sekolah.NibURL != nil && *sekolah.NibURL != "" {
		url, err := s.storage.GeneratePresignedGetURL(ctx, *sekolah.NibURL, getExpiry)
		if err == nil {
			result["nib_url"] = url
		}
	}
	if sekolah.SkPendirianURL != nil && *sekolah.SkPendirianURL != "" {
		url, err := s.storage.GeneratePresignedGetURL(ctx, *sekolah.SkPendirianURL, getExpiry)
		if err == nil {
			result["sk_pendirian_url"] = url
		}
	}
	if sekolah.SiopURL != nil && *sekolah.SiopURL != "" {
		url, err := s.storage.GeneratePresignedGetURL(ctx, *sekolah.SiopURL, getExpiry)
		if err == nil {
			result["siop_url"] = url
		}
	}

	return result, nil
}

