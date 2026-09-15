package mailer

import (
	"fmt"
	"strings"
)

const OTPExpiryMinutes = 3

// BuildOTPEmail builds minimal modern HTML email with OTP digits in styled boxes.
// Palette: primary #049787, background box #E5FFF9, border #6ee7b7.
// Logo placeholder: replace LOGO_URL with actual hosted image URL.
func BuildOTPEmail(name, otp string) string {
	digits := buildOTPDigits(otp)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Kode Verifikasi SIMAS</title>
</head>
<body style="margin:0;padding:0;background-color:#f9fafb;font-family:'Segoe UI',Arial,sans-serif;">

  <table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;padding:48px 16px;">
    <tr>
      <td align="center">

        <table width="440" cellpadding="0" cellspacing="0">

          <!-- Logo + Wordmark -->
          <tr>
            <td align="center" style="padding-bottom:32px;">
              <!--
                TODO: Ganti LOGO_URL dengan URL gambar logo yang sudah dihosting.
                Rekomendasi: PNG transparan, tinggi 40px, hosted di S3 atau CDN.
                Contoh: https://cdn.simas.com/assets/logo.png
              -->
              <img src="LOGO_URL" alt="SIMAS" height="40"
                style="height:40px;display:block;border:0;outline:none;"
                onerror="this.style.display='none'">
              <p style="margin:10px 0 0;font-size:15px;font-weight:700;color:#049787;letter-spacing:2px;">
                SIMAS
              </p>
            </td>
          </tr>

          <!-- Card -->
          <tr>
            <td style="background:#ffffff;border-radius:16px;border:1px solid #e5e7eb;padding:40px 36px;">

              <!-- Greeting -->
              <p style="margin:0 0 6px;font-size:20px;font-weight:700;color:#111827;">
                Halo, %s
              </p>
              <p style="margin:0 0 32px;font-size:14px;color:#6b7280;line-height:1.7;">
                Masukkan kode verifikasi berikut untuk melanjutkan registrasi akun Wali Murid Anda.
              </p>

              <!-- OTP Label -->
              <p style="margin:0 0 12px;font-size:11px;font-weight:700;color:#049787;letter-spacing:3px;text-transform:uppercase;text-align:center;">
                Kode Verifikasi
              </p>

              <!-- OTP Digits -->
              <table cellpadding="0" cellspacing="0" align="center" style="margin:0 auto;">
                <tr>
                  %s
                </tr>
              </table>

              <!-- Expiry -->
              <p style="margin:20px 0 0;font-size:13px;color:#6b7280;text-align:center;">
                Kode berlaku selama
                <strong style="color:#049787;">%d menit</strong>
                dan hanya dapat digunakan satu kali.
              </p>

              <!-- Divider -->
              <hr style="border:none;border-top:1px solid #f3f4f6;margin:32px 0;">

              <!-- Security note -->
              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>
                  <td style="background:#FFFBEB;border-radius:8px;padding:14px 18px;">
                    <p style="margin:0;font-size:12px;color:#78350F;line-height:1.6;">
                      Jangan bagikan kode ini kepada siapapun, termasuk pihak yang mengaku dari SIMAS. Jika Anda tidak melakukan registrasi, abaikan email ini.
                    </p>
                  </td>
                </tr>
              </table>

            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td align="center" style="padding-top:28px;">
              <p style="margin:0;font-size:12px;color:#9ca3af;line-height:1.8;">
                Email otomatis dari sistem SIMAS &mdash; mohon tidak dibalas.<br>
                &copy; 2026 SIMAS. Seluruh hak cipta dilindungi.
              </p>
            </td>
          </tr>

        </table>

      </td>
    </tr>
  </table>

</body>
</html>`, name, digits, OTPExpiryMinutes)
}

// buildOTPDigits converts each OTP digit into an individually styled table cell.
func buildOTPDigits(otp string) string {
	cells := make([]string, 0, len(otp))
	for _, d := range otp {
		cells = append(cells, fmt.Sprintf(
			`<td style="width:52px;height:60px;background:#E5FFF9;border:2px solid #049787;border-radius:10px;text-align:center;vertical-align:middle;font-size:32px;font-weight:800;color:#049787;font-family:'Courier New',Courier,monospace;">%c</td>`,
			d,
		))
	}
	spacer := `<td style="width:8px;"></td>`
	return strings.Join(cells, spacer)
}

// ---- Sekolah Notifications ----

// BuildSekolahPengajuanNotifEmail notifikasi ke Super Admin ada sekolah baru menunggu verifikasi
func BuildSekolahPengajuanNotifEmail(sekolahNama, sekolahEmail, sekolahJenjang string) string {
	jenjangLabel := map[string]string{
		"sd_mi": "SD / MI",
		"smp_mts": "SMP / MTS",
		"sma_smk_ma_mak": "SMA / SMK / MA / MAK",
	}[sekolahJenjang]
	if jenjangLabel == "" {
		jenjangLabel = sekolahJenjang
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><title>Pengajuan Sekolah Baru - SIMAS</title></head>
<body style="margin:0;padding:0;background-color:#f9fafb;font-family:'Segoe UI',Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;padding:48px 16px;">
  <tr><td align="center">
    <table width="440" cellpadding="0" cellspacing="0">
      <tr><td align="center" style="padding-bottom:32px;">
        <p style="margin:0;font-size:15px;font-weight:700;color:#049787;letter-spacing:2px;">SIMAS</p>
      </td></tr>
      <tr><td style="background:#ffffff;border-radius:16px;border:1px solid #e5e7eb;padding:40px 36px;">
        <p style="margin:0 0 6px;font-size:20px;font-weight:700;color:#111827;">Pengajuan Sekolah Baru</p>
        <p style="margin:0 0 24px;font-size:14px;color:#6b7280;line-height:1.7;">Terdapat satuan pendidikan baru yang mengajukan pendaftaran dan menunggu verifikasi Anda.</p>
        <table width="100%%" cellpadding="0" cellspacing="0" style="background:#F0FDFA;border-radius:8px;padding:14px 18px;margin-bottom:24px;">
          <tr><td>
            <p style="margin:0 0 4px;font-size:12px;color:#6b7280;">Nama Sekolah</p>
            <p style="margin:0 0 16px;font-size:14px;font-weight:600;color:#111827;">%s</p>
            <p style="margin:0 0 4px;font-size:12px;color:#6b7280;">Email</p>
            <p style="margin:0 0 16px;font-size:14px;font-weight:600;color:#111827;">%s</p>
            <p style="margin:0 0 4px;font-size:12px;color:#6b7280;">Jenjang</p>
            <p style="margin:0;font-size:14px;font-weight:600;color:#111827;">%s</p>
          </td></tr>
        </table>
        <p style="margin:0;font-size:13px;color:#6b7280;text-align:center;">Silakan login ke dashboard SIMAS untuk melakukan verifikasi.</p>
      </td></tr>
      <tr><td align="center" style="padding-top:28px;">
        <p style="margin:0;font-size:12px;color:#9ca3af;line-height:1.8;">Email otomatis dari sistem SIMAS &mdash; mohon tidak dibalas.<br>&copy; 2026 SIMAS. Seluruh hak cipta dilindungi.</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`, sekolahNama, sekolahEmail, jenjangLabel)
}

// BuildSekolahDiterimaEmail notifikasi ke sekolah bahwa pengajuan disetujui
func BuildSekolahDiterimaEmail(sekolahNama string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><title>Pengajuan Diterima - SIMAS</title></head>
<body style="margin:0;padding:0;background-color:#f9fafb;font-family:'Segoe UI',Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;padding:48px 16px;">
  <tr><td align="center">
    <table width="440" cellpadding="0" cellspacing="0">
      <tr><td align="center" style="padding-bottom:32px;">
        <p style="margin:0;font-size:15px;font-weight:700;color:#049787;letter-spacing:2px;">SIMAS</p>
      </td></tr>
      <tr><td style="background:#ffffff;border-radius:16px;border:1px solid #e5e7eb;padding:40px 36px;">
        <p style="margin:0 0 6px;font-size:20px;font-weight:700;color:#111827;">Pengajuan Diterima</p>
        <p style="margin:0 0 24px;font-size:14px;color:#6b7280;line-height:1.7;">Selamat, pengajuan sekolah <strong>%s</strong> telah disetujui. Sekolah Anda sekarang berstatus <strong>aktif</strong> dan dapat menggunakan layanan SIMAS.</p>
        <table width="100%%" cellpadding="0" cellspacing="0" style="background:#F0FDFA;border-radius:8px;padding:14px 18px;margin-bottom:24px;">
          <tr><td>
            <p style="margin:0;font-size:12px;color:#6b7280;text-align:center;">Status: <strong style="color:#049787;">AKTIF</strong></p>
          </td></tr>
        </table>
        <p style="margin:0;font-size:13px;color:#6b7280;text-align:center;">Anda akan menerima email terpisah berisi kredensial akun Administrator Sekolah.</p>
      </td></tr>
      <tr><td align="center" style="padding-top:28px;">
        <p style="margin:0;font-size:12px;color:#9ca3af;line-height:1.8;">Email otomatis dari sistem SIMAS &mdash; mohon tidak dibalas.<br>&copy; 2026 SIMAS. Seluruh hak cipta dilindungi.</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`, sekolahNama)
}

// BuildSekolahDitolakEmail notifikasi ke sekolah bahwa pengajuan ditolak
func BuildSekolahDitolakEmail(sekolahNama, alasan string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><title>Pengajuan Ditolak - SIMAS</title></head>
<body style="margin:0;padding:0;background-color:#f9fafb;font-family:'Segoe UI',Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;padding:48px 16px;">
  <tr><td align="center">
    <table width="440" cellpadding="0" cellspacing="0">
      <tr><td align="center" style="padding-bottom:32px;">
        <p style="margin:0;font-size:15px;font-weight:700;color:#049787;letter-spacing:2px;">SIMAS</p>
      </td></tr>
      <tr><td style="background:#ffffff;border-radius:16px;border:1px solid #e5e7eb;padding:40px 36px;">
        <p style="margin:0 0 6px;font-size:20px;font-weight:700;color:#111827;">Pengajuan Ditolak</p>
        <p style="margin:0 0 24px;font-size:14px;color:#6b7280;line-height:1.7;">Mohon maaf, pengajuan sekolah <strong>%s</strong> belum dapat disetujui.</p>
        <table width="100%%" cellpadding="0" cellspacing="0" style="background:#FEF2F2;border-radius:8px;padding:14px 18px;margin-bottom:24px;">
          <tr><td>
            <p style="margin:0 0 4px;font-size:12px;color:#6b7280;">Alasan Penolakan</p>
            <p style="margin:0;font-size:14px;font-weight:600;color:#991B1B;">%s</p>
          </td></tr>
        </table>
        <p style="margin:0;font-size:13px;color:#6b7280;text-align:center;">Anda dapat mengajukan ulang dengan memperbaiki data dan dokumen sesuai catatan di atas.</p>
      </td></tr>
      <tr><td align="center" style="padding-top:28px;">
        <p style="margin:0;font-size:12px;color:#9ca3af;line-height:1.8;">Email otomatis dari sistem SIMAS &mdash; mohon tidak dibalas.<br>&copy; 2026 SIMAS. Seluruh hak cipta dilindungi.</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`, sekolahNama, alasan)
}

// BuildKredensialEmail email kredensial login untuk admin_sekolah / tenaga_pendidik
func BuildKredensialEmail(nama, role, email, password, loginURL string) string {
	roleLabel := map[string]string{
		"admin_sekolah":   "Administrator Sekolah",
		"tenaga_pendidik": "Tenaga Pendidik",
	}[role]
	if roleLabel == "" {
		roleLabel = role
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><title>Kredensial Akun - SIMAS</title></head>
<body style="margin:0;padding:0;background-color:#f9fafb;font-family:'Segoe UI',Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;padding:48px 16px;">
  <tr><td align="center">
    <table width="440" cellpadding="0" cellspacing="0">
      <tr><td align="center" style="padding-bottom:32px;">
        <p style="margin:0;font-size:15px;font-weight:700;color:#049787;letter-spacing:2px;">SIMAS</p>
      </td></tr>
      <tr><td style="background:#ffffff;border-radius:16px;border:1px solid #e5e7eb;padding:40px 36px;">
        <p style="margin:0 0 6px;font-size:20px;font-weight:700;color:#111827;">Kredensial Akun Anda</p>
        <p style="margin:0 0 24px;font-size:14px;color:#6b7280;line-height:1.7;">Selamat datang, <strong>%s</strong>. Berikut adalah kredensial akun %s Anda.</p>
        <table width="100%%" cellpadding="0" cellspacing="0" style="background:#F0FDFA;border-radius:8px;padding:14px 18px;margin-bottom:24px;">
          <tr><td>
            <p style="margin:0 0 4px;font-size:12px;color:#6b7280;">Email</p>
            <p style="margin:0 0 16px;font-size:14px;font-weight:600;color:#111827;">%s</p>
            <p style="margin:0 0 4px;font-size:12px;color:#6b7280;">Password Sementara</p>
            <p style="margin:0;font-size:14px;font-weight:600;color:#049787;font-family:'Courier New',monospace;letter-spacing:1px;">%s</p>
          </td></tr>
        </table>
        <table width="100%%" cellpadding="0" cellspacing="0" style="margin-bottom:24px;">
          <tr><td align="center">
            <a href="%s" style="display:inline-block;background:#049787;color:#ffffff;text-decoration:none;padding:12px 32px;border-radius:8px;font-size:14px;font-weight:600;">Login Sekarang</a>
          </td></tr>
        </table>
        <table width="100%%" cellpadding="0" cellspacing="0">
          <tr><td style="background:#FFFBEB;border-radius:8px;padding:14px 18px;">
            <p style="margin:0;font-size:12px;color:#78350F;line-height:1.6;">
              <strong>Penting:</strong> Password di atas bersifat sementara. Anda wajib mengganti password saat login pertama kali. Jangan bagikan kredensial ini kepada siapapun.
            </p>
          </td></tr>
        </table>
      </td></tr>
      <tr><td align="center" style="padding-top:28px;">
        <p style="margin:0;font-size:12px;color:#9ca3af;line-height:1.8;">Email otomatis dari sistem SIMAS &mdash; mohon tidak dibalas.<br>&copy; 2026 SIMAS. Seluruh hak cipta dilindungi.</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`, nama, roleLabel, email, password, loginURL)
}
