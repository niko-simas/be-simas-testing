package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"simas-backend/internal/middleware"
	"simas-backend/internal/model"
	"simas-backend/internal/service"
)

type PendidikHandler struct {
	pendidikService *service.PendidikService
	sekolahService  *service.SekolahService
}

func NewPendidikHandler(pendidikService *service.PendidikService, sekolahService *service.SekolahService) *PendidikHandler {
	return &PendidikHandler{
		pendidikService: pendidikService,
		sekolahService:  sekolahService,
	}
}

func respondPendidikError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: "email_exists", Message: "Email sudah terdaftar"})
	case errors.Is(err, service.ErrPendidikNotFound):
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: "not_found", Message: "Pendidik tidak ditemukan"})
	case errors.Is(err, service.ErrPendidikExists):
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: "pendidik_exists", Message: "NIP atau NUPTK sudah terdaftar"})
	case errors.Is(err, service.ErrJadwalTabrakan):
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: "jadwal_tabrakan", Message: "Jadwal bertabrakan dengan jadwal lain"})
	case errors.Is(err, service.ErrInvalidJadwal):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_jadwal", Message: "Jadwal tidak valid"})
	case errors.Is(err, service.ErrNotWaliKelas):
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: "not_wali_kelas", Message: "Bukan wali kelas"})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: err.Error()})
	}
}

// POST /api/v1/sekolah/saya/pendidik - Admin Sekolah
func (h *PendidikHandler) Tambah(c *gin.Context) {
	var req model.PendidikRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	pendidik, password, err := h.pendidikService.TambahPendidik(c.Request.Context(), user.ID, &req)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"pendidik":     pendidik,
		"temp_password": password,
	})
}

// GET /api/v1/sekolah/saya/pendidik - Admin Sekolah
func (h *PendidikHandler) List(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	status := c.Query("status")
	jabatan := c.Query("jabatan")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Get sekolah milik admin
	sekolah, err := h.sekolahService.GetMySekolah(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: "Tidak dapat menemukan sekolah"})
		return
	}

	pendidiks, total, err := h.pendidikService.ListPendidikBySekolah(c.Request.Context(), sekolah.ID, status, jabatan, search, page, limit)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  pendidiks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GET /api/v1/sekolah/saya/pendidik/:id - Admin Sekolah
func (h *PendidikHandler) Detail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	pendidik, err := h.pendidikService.GetPendidik(c.Request.Context(), id)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	// TODO: Include jadwal
	c.JSON(http.StatusOK, pendidik)
}

// PUT /api/v1/sekolah/saya/pendidik/:id - Admin Sekolah
func (h *PendidikHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	var req model.PendidikUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	pendidik, err := h.pendidikService.UpdatePendidik(c.Request.Context(), id, &req)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, pendidik)
}

// PUT /api/v1/sekolah/saya/pendidik/:id/status - Admin Sekolah
func (h *PendidikHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID tidak valid"})
		return
	}

	var req model.PendidikStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	pendidik, err := h.pendidikService.UpdatePendidikStatus(c.Request.Context(), id, req.Status, req.Alasan)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, pendidik)
}

// POST /api/v1/sekolah/saya/pendidik/:id/jadwal - Admin Sekolah
func (h *PendidikHandler) TambahJadwal(c *gin.Context) {
	pendidikID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID pendidik tidak valid"})
		return
	}

	var req model.PendidikJadwalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	// Get sekolah milik admin
	sekolah, err := h.sekolahService.GetMySekolah(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: "Tidak dapat menemukan sekolah"})
		return
	}

	jadwal, err := h.pendidikService.TambahJadwal(c.Request.Context(), pendidikID, sekolah.ID, &req)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusCreated, jadwal)
}

// GET /api/v1/sekolah/saya/pendidik/:id/jadwal - Admin Sekolah
func (h *PendidikHandler) ListJadwal(c *gin.Context) {
	pendidikID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID pendidik tidak valid"})
		return
	}

	semester := c.Query("semester")

	jadwals, err := h.pendidikService.ListJadwal(c.Request.Context(), pendidikID, semester)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, jadwals)
}

// PUT /api/v1/sekolah/saya/pendidik/:id/jadwal/:jadwal_id - Admin Sekolah
func (h *PendidikHandler) UpdateJadwal(c *gin.Context) {
	jadwalID, err := uuid.Parse(c.Param("jadwal_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID jadwal tidak valid"})
		return
	}

	var req model.PendidikJadwalUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	// Get sekolah milik admin
	sekolah, err := h.sekolahService.GetMySekolah(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: "Tidak dapat menemukan sekolah"})
		return
	}

	jadwal, err := h.pendidikService.UpdateJadwal(c.Request.Context(), jadwalID, sekolah.ID, &req)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, jadwal)
}

// DELETE /api/v1/sekolah/saya/pendidik/:id/jadwal/:jadwal_id - Admin Sekolah
func (h *PendidikHandler) HapusJadwal(c *gin.Context) {
	jadwalID, err := uuid.Parse(c.Param("jadwal_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_id", Message: "ID jadwal tidak valid"})
		return
	}

	// Get sekolah milik admin
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	sekolah, err := h.sekolahService.GetMySekolah(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: "internal_error", Message: "Tidak dapat menemukan sekolah"})
		return
	}

	if err := h.pendidikService.DeleteJadwal(c.Request.Context(), jadwalID, sekolah.ID); err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil dihapus"})
}

// GET /api/v1/pendidik/saya - Tenaga Pendidik
func (h *PendidikHandler) MyProfile(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	pendidik, err := h.pendidikService.MyProfile(c.Request.Context(), user.ID)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, pendidik)
}

// PUT /api/v1/pendidik/saya - Tenaga Pendidik
func (h *PendidikHandler) UpdateMyProfile(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	var req model.PendidikUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: "invalid_request", Message: err.Error()})
		return
	}

	pendidik, err := h.pendidikService.UpdateMyProfile(c.Request.Context(), user.ID, &req)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, pendidik)
}

// GET /api/v1/pendidik/saya/jadwal - Tenaga Pendidik
func (h *PendidikHandler) MyJadwalHarian(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	tanggal := c.Query("tanggal")
	if tanggal == "" {
		tanggal = ""
	}

	jadwals, err := h.pendidikService.MyJadwalHarian(c.Request.Context(), user.ID, tanggal)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, jadwals)
}

// GET /api/v1/pendidik/saya/jadwal/minggu-ini - Tenaga Pendidik
func (h *PendidikHandler) MyJadwalMingguIni(c *gin.Context) {
	user := middleware.InjectUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: "unauthorized", Message: "Tidak terautentikasi"})
		return
	}

	jadwals, err := h.pendidikService.MyJadwalMingguIni(c.Request.Context(), user.ID)
	if err != nil {
		respondPendidikError(c, err)
		return
	}

	c.JSON(http.StatusOK, jadwals)
}
