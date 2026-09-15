package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"simas-backend/internal/auth"
	"simas-backend/internal/infrastructure/mailer"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
)

var (
	ErrPendidikNotFound    = errors.New("pendidik not found")
	ErrPendidikExists      = errors.New("pendidik already exists")
	ErrJadwalTabrakan      = errors.New("jadwal tabrakan detected")
	ErrInvalidJadwal       = errors.New("invalid jadwal data")
	ErrNotWaliKelas        = errors.New("pendidik is not a wali kelas")
)

type PendidikService struct {
	pendidikRepo      *repository.PendidikRepository
	jadwalRepo        *repository.PendidikJadwalRepository
	userRepo          *repository.UserRepository
	sekolahRepo       *repository.SekolahRepository
	mailer            *mailer.SMTPMailer
}

func NewPendidikService(
	pendidikRepo *repository.PendidikRepository,
	jadwalRepo *repository.PendidikJadwalRepository,
	userRepo *repository.UserRepository,
	sekolahRepo *repository.SekolahRepository,
	mailer *mailer.SMTPMailer,
) *PendidikService {
	return &PendidikService{
		pendidikRepo: pendidikRepo,
		jadwalRepo:   jadwalRepo,
		userRepo:     userRepo,
		sekolahRepo:  sekolahRepo,
		mailer:       mailer,
	}
}

func (s *PendidikService) TambahPendidik(ctx context.Context, adminID uuid.UUID, req *model.PendidikRequest) (*model.Pendidik, string, error) {
	// Get sekolah milik admin
	sekolah, err := s.sekolahRepo.FindByAdminID(ctx, adminID)
	if err != nil {
		if errors.Is(err, repository.ErrSekolahNotFound) {
			return nil, "", errors.New("admin tidak memiliki sekolah")
		}
		return nil, "", fmt.Errorf("find sekolah: %w", err)
	}

	// Validasi email unik di users
	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, "", fmt.Errorf("check email: %w", err)
	}
	if existingUser != nil {
		return nil, "", ErrEmailAlreadyExists
	}

	// Validasi NIP unik
	if req.Nip != nil && *req.Nip != "" {
		_, err = s.pendidikRepo.FindByNip(ctx, *req.Nip)
		if err == nil {
			return nil, "", ErrPendidikExists
		}
		if !errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, "", fmt.Errorf("check nip: %w", err)
		}
	}

	// Validasi NUPTK unik
	if req.Nuptk != nil && *req.Nuptk != "" {
		_, err = s.pendidikRepo.FindByNuptk(ctx, *req.Nuptk)
		if err == nil {
			return nil, "", ErrPendidikExists
		}
		if !errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, "", fmt.Errorf("check nuptk: %w", err)
		}
	}

	// Generate password random
	password, err := generateRandomPassword()
	if err != nil {
		return nil, "", fmt.Errorf("generate password: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &model.User{
		Email:             req.Email,
		Name:              req.NamaLengkap,
		Provider:          "email",
		PasswordHash:      &hash,
		Role:              "tenaga_pendidik",
		AccountStatus:     "active",
		OnboardingStatus:  "completed",
		EmailVerified:     true,
		IsPasswordUpdated: false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return nil, "", ErrEmailAlreadyExists
		}
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	// Parse tanggal_lahir
	var tanggalLahir *time.Time
	if req.TanggalLahir != nil && *req.TanggalLahir != "" {
		t, err := time.Parse("2006-01-02", *req.TanggalLahir)
		if err != nil {
			return nil, "", errors.New("invalid tanggal_lahir format, expected YYYY-MM-DD")
		}
		tanggalLahir = &t
	}

	pendidik := &model.Pendidik{
		UserID:            user.ID,
		SekolahID:         sekolah.ID,
		Nip:               req.Nip,
		Nuptk:             req.Nuptk,
		NamaLengkap:       req.NamaLengkap,
		IsWaliKelas:       req.IsWaliKelas,
		JenisKelamin:      req.JenisKelamin,
		TempatLahir:       req.TempatLahir,
		TanggalLahir:      tanggalLahir,
		Agama:             req.Agama,
		Alamat:            req.Alamat,
		NoTelepon:         req.NoTelepon,
		Email:             &req.Email,
		Kewarganegaraan:   req.Kewarganegaraan,
		StatusKepegawaian: req.StatusKepegawaian,
		Jabatan:           req.Jabatan,
		MataPelajaran:     req.MataPelajaran,
		KelasYangDiampu:   req.KelasYangDiampu,
		Status:            "aktif",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.pendidikRepo.Create(ctx, pendidik); err != nil {
		// Rollback user creation
		_ = s.userRepo.Update(ctx, &model.User{ID: user.ID}) // Best effort rollback
		return nil, "", fmt.Errorf("create pendidik: %w", err)
	}

	// Update jumlah_pendidik di sekolah
	sekolah.JumlahPendidik++
	_ = s.sekolahRepo.Update(ctx, sekolah)

	// Send credentials email to pendidik (async)
	loginURL := "https://simas.id/login"
	credBody := mailer.BuildKredensialEmail(req.NamaLengkap, "tenaga_pendidik", req.Email, password, loginURL)
	s.mailer.SendAsync(req.Email, "[SIMAS] Kredensial Akun Tenaga Pendidik", credBody)

	return pendidik, password, nil
}

func (s *PendidikService) GetPendidik(ctx context.Context, id uuid.UUID) (*model.Pendidik, error) {
	pendidik, err := s.pendidikRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}
	return pendidik, nil
}

func (s *PendidikService) GetPendidikByUserID(ctx context.Context, userID uuid.UUID) (*model.Pendidik, error) {
	pendidik, err := s.pendidikRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}
	return pendidik, nil
}

func (s *PendidikService) ListPendidikBySekolah(ctx context.Context, sekolahID uuid.UUID, status string, jabatan string, search string, page int, limit int) ([]model.Pendidik, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.pendidikRepo.ListBySekolahID(ctx, sekolahID, status, jabatan, search, page, limit)
}

func (s *PendidikService) UpdatePendidik(ctx context.Context, pendidikID uuid.UUID, req *model.PendidikUpdateRequest) (*model.Pendidik, error) {
	pendidik, err := s.pendidikRepo.FindByID(ctx, pendidikID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}

	// Update fields if provided
	if req.NamaLengkap != nil {
		pendidik.NamaLengkap = *req.NamaLengkap
	}
	if req.IsWaliKelas != nil {
		pendidik.IsWaliKelas = *req.IsWaliKelas
	}
	if req.JenisKelamin != nil {
		pendidik.JenisKelamin = req.JenisKelamin
	}
	if req.TempatLahir != nil {
		pendidik.TempatLahir = req.TempatLahir
	}
	if req.TanggalLahir != nil && *req.TanggalLahir != "" {
		t, err := time.Parse("2006-01-02", *req.TanggalLahir)
		if err != nil {
			return nil, errors.New("invalid tanggal_lahir format, expected YYYY-MM-DD")
		}
		pendidik.TanggalLahir = &t
	}
	if req.Agama != nil {
		pendidik.Agama = req.Agama
	}
	if req.Alamat != nil {
		pendidik.Alamat = req.Alamat
	}
	if req.NoTelepon != nil {
		pendidik.NoTelepon = req.NoTelepon
	}
	if req.Kewarganegaraan != nil {
		pendidik.Kewarganegaraan = *req.Kewarganegaraan
	}
	if req.StatusKepegawaian != nil {
		pendidik.StatusKepegawaian = req.StatusKepegawaian
	}
	if req.Jabatan != nil {
		pendidik.Jabatan = *req.Jabatan
	}
	if req.MataPelajaran != nil {
		pendidik.MataPelajaran = req.MataPelajaran
	}
	if req.KelasYangDiampu != nil {
		pendidik.KelasYangDiampu = req.KelasYangDiampu
	}
	if req.FotoURL != nil {
		pendidik.FotoURL = req.FotoURL
	}

	pendidik.UpdatedAt = time.Now()

	if err := s.pendidikRepo.Update(ctx, pendidik); err != nil {
		return nil, fmt.Errorf("update pendidik: %w", err)
	}
	return pendidik, nil
}

func (s *PendidikService) UpdatePendidikStatus(ctx context.Context, pendidikID uuid.UUID, status string, alasan *string) (*model.Pendidik, error) {
	pendidik, err := s.pendidikRepo.FindByID(ctx, pendidikID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}

	pendidik.Status = status
	pendidik.UpdatedAt = time.Now()

	if err := s.pendidikRepo.Update(ctx, pendidik); err != nil {
		return nil, fmt.Errorf("update pendidik status: %w", err)
	}

	// Update jumlah pendidik di sekolah
	sekolah, err := s.sekolahRepo.FindByID(ctx, pendidik.SekolahID)
	if err == nil {
		count, _ := s.pendidikRepo.CountBySekolahID(ctx, pendidik.SekolahID)
		sekolah.JumlahPendidik = int(count)
		_ = s.sekolahRepo.Update(ctx, sekolah)
	}

	// TODO: Send notification email if deactivated (async)
	_ = alasan

	return pendidik, nil
}

func (s *PendidikService) MyProfile(ctx context.Context, userID uuid.UUID) (*model.Pendidik, error) {
	return s.GetPendidikByUserID(ctx, userID)
}

func (s *PendidikService) UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *model.PendidikUpdateRequest) (*model.Pendidik, error) {
	pendidik, err := s.pendidikRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}
	return s.UpdatePendidik(ctx, pendidik.ID, req)
}

// ---- Jadwal Mengajar ----

func (s *PendidikService) TambahJadwal(ctx context.Context, pendidikID uuid.UUID, sekolahID uuid.UUID, req *model.PendidikJadwalRequest) (*model.PendidikJadwal, error) {
	// Validate pendidik exists and belongs to sekolah
	pendidik, err := s.pendidikRepo.FindByID(ctx, pendidikID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}
	if pendidik.SekolahID != sekolahID {
		return nil, errors.New("pendidik tidak termasuk dalam sekolah ini")
	}

	// Check tabrakan jadwal
	tabrakan, err := s.jadwalRepo.CheckTabrakan(ctx, sekolahID, req.Hari, req.JamMulai, req.JamSelesai, nil)
	if err != nil {
		return nil, fmt.Errorf("check tabrakan: %w", err)
	}
	if tabrakan {
		return nil, ErrJadwalTabrakan
	}

	jadwal := &model.PendidikJadwal{
		PendidikID:  pendidikID,
		SekolahID:   sekolahID,
		KelasID:     req.KelasID,
		MapelID:     req.MapelID,
		Hari:        req.Hari,
		JamMulai:    req.JamMulai,
		JamSelesai:  req.JamSelesai,
		Semester:    req.Semester,
		TahunAjaran: req.TahunAjaran,
		Status:      "aktif",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.jadwalRepo.Create(ctx, jadwal); err != nil {
		return nil, fmt.Errorf("create jadwal: %w", err)
	}
	return jadwal, nil
}

func (s *PendidikService) ListJadwal(ctx context.Context, pendidikID uuid.UUID, semester string) ([]model.PendidikJadwal, error) {
	return s.jadwalRepo.ListByPendidikID(ctx, pendidikID, semester)
}

func (s *PendidikService) UpdateJadwal(ctx context.Context, jadwalID uuid.UUID, sekolahID uuid.UUID, req *model.PendidikJadwalUpdateRequest) (*model.PendidikJadwal, error) {
	jadwal, err := s.jadwalRepo.FindByID(ctx, jadwalID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrInvalidJadwal
		}
		return nil, fmt.Errorf("find jadwal: %w", err)
	}

	if jadwal.SekolahID != sekolahID {
		return nil, errors.New("jadwal tidak termasuk dalam sekolah ini")
	}

	// Check tabrakan jika ada perubahan hari/jam
	if req.Hari != nil || req.JamMulai != nil || req.JamSelesai != nil {
		hari := jadwal.Hari
		if req.Hari != nil {
			hari = *req.Hari
		}
		jamMulai := jadwal.JamMulai
		if req.JamMulai != nil {
			jamMulai = *req.JamMulai
		}
		jamSelesai := jadwal.JamSelesai
		if req.JamSelesai != nil {
			jamSelesai = *req.JamSelesai
		}

		tabrakan, err := s.jadwalRepo.CheckTabrakan(ctx, sekolahID, hari, jamMulai, jamSelesai, &jadwalID)
		if err != nil {
			return nil, fmt.Errorf("check tabrakan: %w", err)
		}
		if tabrakan {
			return nil, ErrJadwalTabrakan
		}
	}

	if req.Hari != nil {
		jadwal.Hari = *req.Hari
	}
	if req.JamMulai != nil {
		jadwal.JamMulai = *req.JamMulai
	}
	if req.JamSelesai != nil {
		jadwal.JamSelesai = *req.JamSelesai
	}
	if req.KelasID != nil {
		jadwal.KelasID = req.KelasID
	}
	if req.MapelID != nil {
		jadwal.MapelID = req.MapelID
	}
	if req.Semester != nil {
		jadwal.Semester = req.Semester
	}
	if req.TahunAjaran != nil {
		jadwal.TahunAjaran = req.TahunAjaran
	}
	jadwal.UpdatedAt = time.Now()

	if err := s.jadwalRepo.Update(ctx, jadwal); err != nil {
		return nil, fmt.Errorf("update jadwal: %w", err)
	}
	return jadwal, nil
}

func (s *PendidikService) DeleteJadwal(ctx context.Context, jadwalID uuid.UUID, sekolahID uuid.UUID) error {
	jadwal, err := s.jadwalRepo.FindByID(ctx, jadwalID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return ErrInvalidJadwal
		}
		return fmt.Errorf("find jadwal: %w", err)
	}
	if jadwal.SekolahID != sekolahID {
		return errors.New("jadwal tidak termasuk dalam sekolah ini")
	}
	return s.jadwalRepo.Delete(ctx, jadwalID)
}

func (s *PendidikService) MyJadwalHarian(ctx context.Context, userID uuid.UUID, tanggal string) ([]model.PendidikJadwal, error) {
	pendidik, err := s.pendidikRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}

	// Parse tanggal to get day name in Indonesian
	t, err := time.Parse("2006-01-02", tanggal)
	if err != nil {
		return nil, errors.New("invalid tanggal format, expected YYYY-MM-DD")
	}

	hariMap := map[time.Weekday]string{
		time.Monday:    "senin",
		time.Tuesday:   "selasa",
		time.Wednesday: "rabu",
		time.Thursday:  "kamis",
		time.Friday:    "jumat",
		time.Saturday:  "sabtu",
		time.Sunday:    "minggu",
	}
	hari := hariMap[t.Weekday()]

	// Get all jadwal for this pendidik and filter by hari
	jadwals, err := s.jadwalRepo.ListByPendidikID(ctx, pendidik.ID, "")
	if err != nil {
		return nil, fmt.Errorf("list jadwal: %w", err)
	}

	var result []model.PendidikJadwal
	for _, j := range jadwals {
		if j.Hari == hari && j.Status == "aktif" {
			result = append(result, j)
		}
	}
	return result, nil
}

func (s *PendidikService) MyJadwalMingguIni(ctx context.Context, userID uuid.UUID) (map[string][]model.PendidikJadwal, error) {
	pendidik, err := s.pendidikRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrPendidikNotFound) {
			return nil, ErrPendidikNotFound
		}
		return nil, fmt.Errorf("find pendidik: %w", err)
	}

	jadwals, err := s.jadwalRepo.ListByPendidikID(ctx, pendidik.ID, "")
	if err != nil {
		return nil, fmt.Errorf("list jadwal: %w", err)
	}

	result := make(map[string][]model.PendidikJadwal)
	hariList := []string{"senin", "selasa", "rabu", "kamis", "jumat", "sabtu", "minggu"}
	for _, h := range hariList {
		result[h] = []model.PendidikJadwal{}
	}

	for _, j := range jadwals {
		if j.Status == "aktif" {
			result[j.Hari] = append(result[j.Hari], j)
		}
	}
	return result, nil
}
