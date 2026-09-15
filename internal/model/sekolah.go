package model

import (
	"time"

	"github.com/google/uuid"
)

// State machine sekolah:
// pengajuan -> menunggu_verifikasi -> aktif
//                 -> ditolak (bisa pengajuan ulang)

type Sekolah struct {
	ID                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Nama                  string     `gorm:"type:varchar(255);not null" json:"nama"`
	Npsn                  *string    `gorm:"type:varchar(20);unique" json:"npsn,omitempty"`
	Jenjang               string     `gorm:"type:varchar(20);not null" json:"jenjang"`
	Status                string     `gorm:"type:varchar(30);not null;default:'pengajuan'" json:"status"`
	Alamat                *string    `gorm:"type:text" json:"alamat,omitempty"`
	ProvinsiID            *int       `gorm:"column:provinsi_id" json:"provinsi_id,omitempty"`
	KabupatenID           *int       `gorm:"column:kabupaten_id" json:"kabupaten_id,omitempty"`
	KecamatanID           *int       `gorm:"column:kecamatan_id" json:"kecamatan_id,omitempty"`
	DesaID                *int       `gorm:"column:desa_id" json:"desa_id,omitempty"`
	KodePos               *string    `gorm:"type:varchar(10)" json:"kode_pos,omitempty"`
	Latitude              *float64   `gorm:"type:decimal(10,8)" json:"latitude,omitempty"`
	Longitude             *float64   `gorm:"type:decimal(11,8)" json:"longitude,omitempty"`
	Email                 string     `gorm:"type:varchar(255);unique;not null" json:"email"`
	Telepon               *string    `gorm:"type:varchar(20)" json:"telepon,omitempty"`
	Website               *string    `gorm:"type:varchar(255)" json:"website,omitempty"`
	LogoURL               *string    `gorm:"type:text" json:"logo_url,omitempty"`
	AktaPendirianURL      *string    `gorm:"type:text" json:"akta_pendirian_url,omitempty"`
	NibURL                *string    `gorm:"type:text" json:"nib_url,omitempty"`
	SkPendirianURL        *string    `gorm:"type:text" json:"sk_pendirian_url,omitempty"`
	SiopURL               *string    `gorm:"type:text" json:"siop_url,omitempty"`
	Yayasan               *string    `gorm:"type:varchar(255)" json:"yayasan,omitempty"`
	KepalaSekolahNama     *string    `gorm:"type:varchar(255)" json:"kepala_sekolah_nama,omitempty"`
	KepalaSekolahNip      *string    `gorm:"type:varchar(30)" json:"kepala_sekolah_nip,omitempty"`
	JumlahSiswa           int        `gorm:"type:int;default:0" json:"jumlah_siswa"`
	JumlahPendidik        int        `gorm:"type:int;default:0" json:"jumlah_pendidik"`
	TanggalBerdiri        *time.Time `gorm:"type:date" json:"tanggal_berdiri,omitempty"`
	SkPendirianNomor      *string    `gorm:"type:varchar(100)" json:"sk_pendirian,omitempty"`
	TrackingCode          string     `gorm:"type:varchar(20);uniqueIndex;not null" json:"tracking_code"`
	AdminSekolahID        *uuid.UUID `gorm:"type:uuid" json:"admin_sekolah_id,omitempty"`
	CreatedAt             time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

func (Sekolah) TableName() string { return "sekolah" }

type SekolahStateLog struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SekolahID    uuid.UUID  `gorm:"type:uuid;not null" json:"sekolah_id"`
	StatusLama   *string    `gorm:"type:varchar(30)" json:"status_lama,omitempty"`
	StatusBaru   string     `gorm:"type:varchar(30);not null" json:"status_baru"`
	Catatan      *string    `gorm:"type:text" json:"catatan,omitempty"`
	DiprosesOleh *uuid.UUID `gorm:"type:uuid" json:"diproses_oleh,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (SekolahStateLog) TableName() string { return "sekolah_state_log" }

// SekolahRequest — Step 1: kirim data sekolah (tanpa file)
type SekolahRequest struct {
	Nama              string   `json:"nama" form:"nama" binding:"required,max=255"`
	Npsn              *string  `json:"npsn" form:"npsn"`
	Jenjang           string   `json:"jenjang" form:"jenjang" binding:"required,oneof=sd_mi smp_mts sma_smk_ma_mak"`
	Alamat            *string  `json:"alamat" form:"alamat"`
	ProvinsiID        *int     `json:"provinsi_id" form:"provinsi_id"`
	KabupatenID       *int     `json:"kabupaten_id" form:"kabupaten_id"`
	KecamatanID       *int     `json:"kecamatan_id" form:"kecamatan_id"`
	DesaID            *int     `json:"desa_id" form:"desa_id"`
	KodePos           *string  `json:"kode_pos" form:"kode_pos"`
	Latitude          *float64 `json:"latitude" form:"latitude"`
	Longitude         *float64 `json:"longitude" form:"longitude"`
	Email             string   `json:"email" form:"email" binding:"required,email,max=255"`
	Telepon           *string  `json:"telepon" form:"telepon"`
	Website           *string  `json:"website" form:"website"`
	Yayasan           *string  `json:"yayasan" form:"yayasan"`
	KepalaSekolahNama *string  `json:"kepala_sekolah_nama" form:"kepala_sekolah_nama"`
	KepalaSekolahNip  *string  `json:"kepala_sekolah_nip" form:"kepala_sekolah_nip"`
	TanggalBerdiri    *string  `json:"tanggal_berdiri" form:"tanggal_berdiri"`
	SkPendirianNomor  *string  `json:"sk_pendirian" form:"sk_pendirian"`
}

// SekolahDokumenRequest — Step 2: upload 4 file legalitas (multipart/form-data)
type SekolahDokumenRequest struct {
	AktaPendirian *string `json:"akta_pendirian,omitempty" form:"akta_pendirian"`
	Nib           *string `json:"nib,omitempty" form:"nib"`
	SkPendirian   *string `json:"sk_pendirian,omitempty" form:"sk_pendirian"`
	Siop          *string `json:"siop,omitempty" form:"siop"`
}

type SekolahVerifikasiRequest struct {
	StatusBaru string  `json:"status_baru" binding:"required,oneof=menunggu_verifikasi aktif ditolak"`
	Catatan    *string `json:"catatan"`
}

type SekolahUpdateRequest struct {
	Nama              *string  `json:"nama"`
	Npsn              *string  `json:"npsn"`
	Jenjang           *string  `json:"jenjang" binding:"omitempty,oneof=sd_mi smp_mts sma_smk_ma_mak"`
	Alamat            *string  `json:"alamat"`
	ProvinsiID        *int     `json:"provinsi_id"`
	KabupatenID       *int     `json:"kabupaten_id"`
	KecamatanID       *int     `json:"kecamatan_id"`
	DesaID            *int     `json:"desa_id"`
	KodePos           *string  `json:"kode_pos"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	Telepon           *string  `json:"telepon"`
	Website           *string  `json:"website"`
	LogoURL           *string  `json:"logo_url"`
	Yayasan           *string  `json:"yayasan"`
	KepalaSekolahNama *string  `json:"kepala_sekolah_nama"`
	KepalaSekolahNip  *string  `json:"kepala_sekolah_nip"`
	TanggalBerdiri    *string  `json:"tanggal_berdiri"`
	SkPendirianNomor  *string  `json:"sk_pendirian"`
}

// SekolahProgressResponse — untuk public tracking tanpa auth
type SekolahProgressResponse struct {
	TrackingCode  string     `json:"tracking_code"`
	Nama          string     `json:"nama"`
	Status        string     `json:"status"`
	StatusLabel   string     `json:"status_label"`
	Email         string     `json:"email"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StateLogs     []SekolahStateLogSummary `json:"state_logs,omitempty"`
}

type SekolahStateLogSummary struct {
	StatusBaru string     `json:"status_baru"`
	StatusLabel string    `json:"status_label"`
	Catatan    *string    `json:"catatan,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
