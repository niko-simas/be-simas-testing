package model

import (
	"time"

	"github.com/google/uuid"
)

type Pendidik struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	SekolahID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	Nip               *string    `gorm:"type:varchar(30);uniqueIndex" json:"nip,omitempty"`
	Nuptk             *string    `gorm:"type:varchar(30);uniqueIndex" json:"nuptk,omitempty"`
	NamaLengkap       string     `gorm:"type:varchar(255);not null" json:"nama_lengkap"`
	IsWaliKelas       bool       `gorm:"type:boolean;default:false" json:"is_wali_kelas"`
	JenisKelamin      *string    `gorm:"type:varchar(10)" json:"jenis_kelamin,omitempty"`
	TempatLahir       *string    `gorm:"type:varchar(100)" json:"tempat_lahir,omitempty"`
	TanggalLahir      *time.Time `gorm:"type:date" json:"tanggal_lahir,omitempty"`
	Agama             *string    `gorm:"type:varchar(20)" json:"agama,omitempty"`
	Alamat            *string    `gorm:"type:text" json:"alamat,omitempty"`
	NoTelepon         *string    `gorm:"type:varchar(20)" json:"no_telepon,omitempty"`
	Email             *string    `gorm:"type:varchar(255)" json:"email,omitempty"`
	Kewarganegaraan   string     `gorm:"type:varchar(10);not null;default:'WNI'" json:"kewarganegaraan"`
	StatusKepegawaian *string    `gorm:"type:varchar(30)" json:"status_kepegawaian,omitempty"`
	Jabatan           string     `gorm:"type:varchar(50);not null" json:"jabatan"`
	MataPelajaran     *string    `gorm:"type:varchar(100)" json:"mata_pelajaran,omitempty"`
	KelasYangDiampu   *string    `gorm:"type:varchar(100)" json:"kelas_yang_diampu,omitempty"`
	FotoURL           *string    `gorm:"type:text" json:"foto_url,omitempty"`
	Status            string     `gorm:"type:varchar(20);not null;default:'aktif'" json:"status"`
	CreatedAt         time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

func (Pendidik) TableName() string { return "pendidik" }

type PendidikJadwal struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	PendidikID   uuid.UUID `gorm:"type:uuid;not null;index" json:"pendidik_id"`
	SekolahID    uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	KelasID      *uuid.UUID `gorm:"type:uuid" json:"kelas_id,omitempty"`
	MapelID      *uuid.UUID `gorm:"type:uuid" json:"mapel_id,omitempty"`
	Hari         string    `gorm:"type:varchar(10);not null" json:"hari"`
	JamMulai     string    `gorm:"type:varchar(5);not null" json:"jam_mulai"`
	JamSelesai   string    `gorm:"type:varchar(5);not null" json:"jam_selesai"`
	Semester     *string   `gorm:"type:varchar(20)" json:"semester,omitempty"`
	TahunAjaran  *string   `gorm:"type:varchar(20)" json:"tahun_ajaran,omitempty"`
	Status       string    `gorm:"type:varchar(20);not null;default:'aktif'" json:"status"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (PendidikJadwal) TableName() string { return "pendidik_jadwal" }

type PendidikRequest struct {
	Nip             *string `json:"nip"`
	Nuptk           *string `json:"nuptk"`
	NamaLengkap     string  `json:"nama_lengkap" binding:"required,max=255"`
	IsWaliKelas     bool    `json:"is_wali_kelas"`
	JenisKelamin    *string `json:"jenis_kelamin" binding:"omitempty,oneof=laki_laki perempuan"`
	TempatLahir     *string `json:"tempat_lahir"`
	TanggalLahir    *string `json:"tanggal_lahir"`
	Agama           *string `json:"agama"`
	Alamat          *string `json:"alamat"`
	NoTelepon       *string `json:"no_telepon"`
	Email           string  `json:"email" binding:"required,email,max=255"`
	Kewarganegaraan string  `json:"kewarganegaraan" binding:"required,oneof=WNI WNA"`
	StatusKepegawaian *string `json:"status_kepegawaian" binding:"omitempty,oneof=pns ptt honorer"`
	Jabatan         string  `json:"jabatan" binding:"required,oneof=guru_mata_pelajaran wali_kelas kepala_sekolah staff_tu"`
	MataPelajaran   *string `json:"mata_pelajaran"`
	KelasYangDiampu *string `json:"kelas_yang_diampu"`
}

type PendidikUpdateRequest struct {
	NamaLengkap       *string `json:"nama_lengkap"`
	IsWaliKelas       *bool   `json:"is_wali_kelas"`
	JenisKelamin      *string `json:"jenis_kelamin" binding:"omitempty,oneof=laki_laki perempuan"`
	TempatLahir       *string `json:"tempat_lahir"`
	TanggalLahir      *string `json:"tanggal_lahir"`
	Agama             *string `json:"agama"`
	Alamat            *string `json:"alamat"`
	NoTelepon         *string `json:"no_telepon"`
	Kewarganegaraan   *string `json:"kewarganegaraan" binding:"omitempty,oneof=WNI WNA"`
	StatusKepegawaian *string `json:"status_kepegawaian" binding:"omitempty,oneof=pns ptt honorer"`
	Jabatan           *string `json:"jabatan" binding:"omitempty,oneof=guru_mata_pelajaran wali_kelas kepala_sekolah staff_tu"`
	MataPelajaran     *string `json:"mata_pelajaran"`
	KelasYangDiampu   *string `json:"kelas_yang_diampu"`
	FotoURL           *string `json:"foto_url"`
}

type PendidikStatusRequest struct {
	Status string  `json:"status" binding:"required,oneof=aktif nonaktif cuti"`
	Alasan *string `json:"alasan"`
}

type PendidikJadwalRequest struct {
	Hari        string    `json:"hari" binding:"required,oneof=senin selasa rabu kamis jumat sabtu minggu"`
	JamMulai    string    `json:"jam_mulai" binding:"required"`
	JamSelesai  string    `json:"jam_selesai" binding:"required"`
	KelasID     *uuid.UUID `json:"kelas_id"`
	MapelID     *uuid.UUID `json:"mapel_id"`
	Semester    *string   `json:"semester"`
	TahunAjaran *string   `json:"tahun_ajaran"`
}

type PendidikJadwalUpdateRequest struct {
	Hari        *string    `json:"hari" binding:"omitempty,oneof=senin selasa rabu kamis jumat sabtu minggu"`
	JamMulai    *string    `json:"jam_mulai"`
	JamSelesai  *string    `json:"jam_selesai"`
	KelasID     *uuid.UUID `json:"kelas_id"`
	MapelID     *uuid.UUID `json:"mapel_id"`
	Semester    *string   `json:"semester"`
	TahunAjaran *string   `json:"tahun_ajaran"`
}
