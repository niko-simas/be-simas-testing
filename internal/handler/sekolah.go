package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"simas-backend/internal/middleware"
	"simas-backend/internal/model"
	"simas-backend/internal/service"
)

type SekolahHandler struct {
	sekolahService *service.SekolahService
}

func NewSekolahHandler(sekolahService *service.SekolahService) *SekolahHandler {
	return &SekolahHandler{sekolahService: sekolahService}
}

func respondSekolahError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSekolahExists):
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: "sekolah_exists", Message: "Sekolah atau email/NPSN sudah terdaftar"})
	case errors.Is(err, service.ErrSekolahNotFound):
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: "not_found", Message: "Sekolah tidak ditemukan"})
	case errors.Is(err, service.ErrInvalidJenjang):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_jenjang", Message: "Jenjang tidak valid"})
	case errors.Is(err, service.ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_status", Message: "Transisi status tidak valid"})
	case errors.Is(err, service.ErrIncompleteLegalitas):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "incomplete_legalitas", Message: "4 dokumen legalitas wajib diupload"})
	case errors.Is(err, service.ErrFileTooLarge):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "file_too_large", Message: "Ukuran file maksimal 3MB"})
	case errors.Is(err, service.ErrInvalidFileFormat):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_file_format", Message: "File harus berformat PDF"})
	case errors.Is(err, service.ErrPengajuanUlangNotAllowed):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "not_allowed", Message: "Pengajuan ulang hanya diperbolehkan untuk status ditolak"})
	case errors.Is(err, service.ErrInvalidTrackingCode):
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: "invalid_tracking", Message: "Kode tracking tidak ditemukan"})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: err.Error()})
	}
}

// PengajuanBaru godoc
// @Summary Step 1: Pengajuan baru sekolah (data saja)
// @Description Submit data sekolah tanpa file. Response mengembalikan sekolah_id dan tracking_code. Step 2: upload dokumen via POST /sekolah/:id/dokumen
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param request body model.SekolahRequest true "Data sekolah"
// @Success 201 {object} map[string]interface{} "{id, tracking_code, status}"
// @Failure 400 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse
// @Router /api/v1/sekolah [post]
func (h *SekolahHandler) PengajuanBaru(c *gin.Context) {
	var req model.SekolahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	sekolah, err := h.sekolahService.PengajuanBaru(c.Request.Context(), &req)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":              sekolah.ID,
		"tracking_code":   sekolah.TrackingCode,
		"status":          sekolah.Status,
		"status_label":    "Pengajuan Diterima",
		"message":         "Simpan tracking_code untuk melihat progress pengajuan. Silakan upload dokumen legalitas di endpoint /sekolah/" + sekolah.ID.String() + "/dokumen",
	})
}

// PengajuanBaruLengkap godoc
// @Summary Pengajuan baru sekolah dengan data + dokumen legalitas sekaligus
// @Description Submit data sekolah + 4 file PDF legalitas via multipart/form-data. Status langsung "menunggu_verifikasi". Field data sama dengan SekolahRequest, file: akta_pendirian, nib, sk_pendirian, siop.
// @Tags Sekolah
// @Accept multipart/form-data
// @Produce json
// @Param nama formData string true "Nama sekolah"
// @Param jenjang formData string true "Jenjang" Enums(sd_mi, smp_mts, sma_smk_ma_mak)
// @Param email formData string true "Email sekolah"
// @Param npsn formData string false "NPSN"
// @Param alamat formData string false "Alamat"
// @Param provinsi_id formData integer false "Provinsi ID"
// @Param kabupaten_id formData integer false "Kabupaten ID"
// @Param kecamatan_id formData integer false "Kecamatan ID"
// @Param desa_id formData integer false "Desa ID"
// @Param kode_pos formData string false "Kode pos"
// @Param latitude formData number false "Latitude"
// @Param longitude formData number false "Longitude"
// @Param telepon formData string false "Telepon"
// @Param website formData string false "Website"
// @Param yayasan formData string false "Yayasan"
// @Param kepala_sekolah_nama formData string false "Nama kepala sekolah"
// @Param kepala_sekolah_nip formData string false "NIP kepala sekolah"
// @Param tanggal_berdiri formData string false "YYYY-MM-DD"
// @Param sk_pendirian formData string false "Nomor SK pendirian"
// @Param akta_pendirian formData file true "Akta Pendirian (PDF, max 3MB)"
// @Param nib formData file true "NIB (PDF, max 3MB)"
// @Param sk_pendirian_file formData file true "SK Pendirian (PDF, max 3MB)"
// @Param siop formData file true "Surat Izin Operasional (PDF, max 3MB)"
// @Success 201 {object} map[string]interface{} "{id, tracking_code, status, status_label, message}"
// @Failure 400 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse
// @Router /api/v1/sekolah/lengkap [post]
func (h *SekolahHandler) PengajuanBaruLengkap(c *gin.Context) {
	var req model.SekolahRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: "Invalid multipart form: " + err.Error()})
		return
	}

	files := make(map[string][]byte)
	requiredFields := []string{"akta_pendirian", "nib", "sk_pendirian", "siop"}

	for _, field := range requiredFields {
		handlers := form.File[field]
		if len(handlers) == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "incomplete_legalitas", Message: fmt.Sprintf("File %s wajib diupload", field)})
			return
		}

		f, err := handlers[0].Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "upload_error", Message: fmt.Sprintf("Gagal membaca file %s", field)})
			return
		}

		buf := new(bytes.Buffer)
		size, err := buf.ReadFrom(f)
		f.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "upload_error", Message: fmt.Sprintf("Gagal membaca file %s", field)})
			return
		}

		if size > 3*1024*1024 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "file_too_large", Message: fmt.Sprintf("File %s melebihi 3MB", field)})
			return
		}

		files[field] = buf.Bytes()
	}

	sekolah, err := h.sekolahService.PengajuanBaruLengkap(c.Request.Context(), &req, files)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            sekolah.ID,
		"tracking_code": sekolah.TrackingCode,
		"status":        sekolah.Status,
		"status_label":  "Menunggu Verifikasi Admin",
		"message":       "Pengajuan berhasil. Sekolah menunggu verifikasi admin.",
	})
}

// UploadDokumen godoc
// @Summary Step 2: Upload 4 dokumen legalitas (form-data)
// @Description Upload 4 file PDF legalitas via multipart/form-data: akta_pendirian, nib, sk_pendirian, siop. Max 3MB per file. Status otomatis berubah ke "menunggu_verifikasi"
// @Tags Sekolah
// @Accept multipart/form-data
// @Produce json
// @Param sekolah_id path string true "Sekolah ID dari step 1"
// @Param akta_pendirian formData file true "Akta Pendirian Yayasan/Badan Hukum (PDF, max 3MB)"
// @Param nib formData file true "Nomor Induk Berusaha (PDF, max 3MB)"
// @Param sk_pendirian formData file true "SK Pendirian Sekolah (PDF, max 3MB)"
// @Param siop formData file true "Surat Izin Operasional (PDF, max 3MB)"
// @Success 200 {object} model.Sekolah
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Router /api/v1/sekolah/{sekolah_id}/dokumen [post]
func (h *SekolahHandler) UploadDokumen(c *gin.Context) {
	sekolahID, err := uuid.Parse(c.Param("sekolah_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID sekolah tidak valid"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: "Invalid multipart form: " + err.Error()})
		return
	}

	files := make(map[string][]byte)
	requiredFields := []string{"akta_pendirian", "nib", "sk_pendirian", "siop"}

	for _, field := range requiredFields {
		handlers := form.File[field]
		if len(handlers) == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "incomplete_legalitas", Message: fmt.Sprintf("File %s wajib diupload", field)})
			return
		}

		f, err := handlers[0].Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "upload_error", Message: fmt.Sprintf("Gagal membaca file %s", field)})
			return
		}

		buf := new(bytes.Buffer)
		size, err := buf.ReadFrom(f)
		f.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "upload_error", Message: fmt.Sprintf("Gagal membaca file %s", field)})
			return
		}

		if size > 3*1024*1024 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "file_too_large", Message: fmt.Sprintf("File %s melebihi 3MB", field)})
			return
		}

		files[field] = buf.Bytes()
	}

	sekolah, err := h.sekolahService.UploadDokumen(c.Request.Context(), sekolahID, files)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            sekolah.ID,
		"tracking_code": sekolah.TrackingCode,
		"status":        sekolah.Status,
		"status_label":  "Menunggu Verifikasi Admin",
		"message":       "Dokumen berhasil diupload. Pengajuan sedang diverifikasi oleh admin SIMAS.",
	})
}

// Progress godoc
// @Summary Track progress pengajuan sekolah (public, no auth)
// @Description Cek status pengajuan sekolah menggunakan tracking_code. Tidak perlu autentikasi.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param tracking_code query string true "Tracking code dari response pengajuan"
// @Success 200 {object} model.SekolahProgressResponse
// @Failure 404 {object} model.ErrorResponse
// @Router /api/v1/sekolah/progress [get]
func (h *SekolahHandler) Progress(c *gin.Context) {
	trackingCode := c.Query("tracking_code")
	if trackingCode == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "missing_tracking", Message: "tracking_code wajib diisi"})
		return
	}

	progress, err := h.sekolahService.GetProgress(c.Request.Context(), trackingCode)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, progress)
}

// List godoc
// @Summary List sekolah dengan filter
// @Description List semua sekolah dengan filter status, search, dan pagination. Super Admin only.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param status query string false "Filter status: pengajuan, menunggu_verifikasi, aktif, nonaktif, ditolak"
// @Param search query string false "Search nama atau NPSN"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "{data[], total, page, limit}"
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Router /api/v1/sekolah [get]
func (h *SekolahHandler) List(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	sekolahs, total, err := h.sekolahService.ListSekolah(c.Request.Context(), status, search, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  sekolahs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Detail godoc
// @Summary Detail sekolah + state log
// @Description Detail sekolah dengan history state log. Super Admin only.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param id path string true "Sekolah ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "{sekolah, state_logs}"
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Router /api/v1/sekolah/{id} [get]
func (h *SekolahHandler) Detail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	sekolah, logs, err := h.sekolahService.GetSekolah(c.Request.Context(), id)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sekolah":    sekolah,
		"state_logs": logs,
	})
}

// DokumenLegalitas godoc
// @Summary Get presigned URL dokumen legalitas
// @Description Generate presigned URL untuk preview/download 4 dokumen legalitas. Super Admin only.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param id path string true "Sekolah ID"
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Router /api/v1/sekolah/{id}/dokumen-legalitas [get]
func (h *SekolahHandler) DokumenLegalitas(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	docs, err := h.sekolahService.GetDokumenLegalitas(c.Request.Context(), id)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, docs)
}

// Verifikasi godoc
// @Summary Verifikasi pengajuan sekolah
// @Description Super Admin setuju/tolak pengajuan sekolah. Setuju: status -> aktif (auto-generate akun admin_sekolah). Tolak: status -> ditolak.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param id path string true "Sekolah ID"
// @Param request body model.SekolahVerifikasiRequest true "Verifikasi request"
// @Security BearerAuth
// @Success 200 {object} model.Sekolah
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Router /api/v1/sekolah/{id}/verifikasi [put]
func (h *SekolahHandler) Verifikasi(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	var req model.SekolahVerifikasiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	sekolah, err := h.sekolahService.Verifikasi(c.Request.Context(), id, user.ID, req.StatusBaru, req.Catatan)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, sekolah)
}

// Update godoc
// @Summary Update data sekolah
// @Description Update data sekolah. Super Admin only.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param id path string true "Sekolah ID"
// @Param request body model.SekolahUpdateRequest true "Update request"
// @Security BearerAuth
// @Success 200 {object} model.Sekolah
// @Failure 400 {object} model.ErrorResponse
// @Router /api/v1/sekolah/{id} [put]
func (h *SekolahHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	var req model.SekolahUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	sekolah, err := h.sekolahService.UpdateSekolah(c.Request.Context(), id, &req)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, sekolah)
}

// MySekolah godoc
// @Summary Get data sekolah milik admin yang login
// @Description Admin Sekolah melihat data sekolahnya sendiri.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Sekolah
// @Failure 401 {object} model.ErrorResponse
// @Router /api/v1/sekolah/saya [get]
func (h *SekolahHandler) MySekolah(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	sekolah, err := h.sekolahService.GetMySekolah(c.Request.Context(), user.ID)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, sekolah)
}

// UpdateMySekolah godoc
// @Summary Update data sekolah milik admin yang login
// @Description Admin Sekolah update data dasar sekolah (alamat, telepon, website, logo, koordinat).
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param request body model.SekolahUpdateRequest true "Update request"
// @Security BearerAuth
// @Success 200 {object} model.Sekolah
// @Failure 401 {object} model.ErrorResponse
// @Router /api/v1/sekolah/saya [put]
func (h *SekolahHandler) UpdateMySekolah(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	var req model.SekolahUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	sekolah, err := h.sekolahService.UpdateMySekolah(c.Request.Context(), user.ID, &req)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, sekolah)
}

// PengajuanUlang godoc
// @Summary Pengajuan ulang setelah ditolak
// @Description Admin Sekolah mengajukan ulang setelah status ditolak. Status kembali ke pengajuan, dokumen harus diupload ulang.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Param request body model.SekolahRequest true "Data sekolah baru"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "{id, tracking_code, status}"
// @Failure 400 {object} model.ErrorResponse
// @Router /api/v1/sekolah/saya/pengajuan-ulang [post]
func (h *SekolahHandler) PengajuanUlang(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	var req model.SekolahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	// Get existing sekolah to get tracking_code
	sekolahExisting, err := h.sekolahService.GetMySekolah(c.Request.Context(), user.ID)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	sekolah, err := h.sekolahService.PengajuanUlang(c.Request.Context(), sekolahExisting.TrackingCode, &req)
	if err != nil {
		respondSekolahError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            sekolah.ID,
		"tracking_code": sekolah.TrackingCode,
		"status":        sekolah.Status,
		"status_label":  "Pengajuan Diterima",
		"message":       "Pengajuan ulang berhasil. Silakan upload dokumen legalitas di endpoint /sekolah/" + sekolah.ID.String() + "/dokumen",
	})
}

// ListJenjang godoc
// @Summary List jenjang pendidikan (master data)
// @Description Public endpoint untuk mendapatkan daftar jenjang pendidikan.
// @Tags Sekolah
// @Accept json
// @Produce json
// @Success 200 {array} map[string]string
// @Router /api/v1/sekolah/jenjang [get]
func (h *SekolahHandler) ListJenjang(c *gin.Context) {
	jenjangs := []gin.H{
		{"id": "sd_mi", "nama": "SD / MI"},
		{"id": "smp_mts", "nama": "SMP / MTS"},
		{"id": "sma_smk_ma_mak", "nama": "SMA / SMK / MA / MAK"},
	}
	c.JSON(http.StatusOK, jenjangs)
}
