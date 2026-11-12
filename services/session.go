package services

import (
	"errors"
	"log"
	"time"

	"nganime-api/models"
	"nganime-api/utils"

	"gorm.io/gorm"
)

// ErrSessionInvalid dikembalikan bila sesi tidak ada, sudah dicabut/kedaluwarsa,
// atau refresh token yang dipakai bukan token terbaru (indikasi token dicuri).
var ErrSessionInvalid = errors.New("session invalid, revoked, or expired")

// GraceWindow: refresh token lama yang baru saja dirotasi masih diterima selama
// jendela ini. Ini menangani dua request refresh yang berlomba (mis. dua tab browser)
// supaya user tidak ter-logout karena race, bukan karena pencurian.
const GraceWindow = 30 * time.Second

func trimUA(ua string) string {
	if len(ua) > 255 {
		return ua[:255]
	}
	return ua
}

// CreateSession membuat sesi baru untuk satu login. Refresh token pertama memakai
// session.CurrentJTI.
func CreateSession(db *gorm.DB, userID uint, userAgent string) (*models.Session, error) {
	jti, err := utils.NewJTI()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	s := &models.Session{
		UserID:     userID,
		CurrentJTI: jti,
		ExpiresAt:  now.Add(utils.RefreshTokenTTL()),
		LastUsedAt: now,
		UserAgent:  trimUA(userAgent),
	}
	if err := db.Create(s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// RotateSession memeriksa refresh token terhadap sesinya lalu menerbitkan JTI baru
// (rolling: umur sesi dihitung ulang). Mengembalikan sesi dengan CurrentJTI terbaru.
//
// Memakai ulang token lama di luar GraceWindow = token kemungkinan dicuri, maka seluruh
// sesi dicabut dan pemilik sah harus login lagi.
func RotateSession(db *gorm.DB, c *utils.RefreshClaims, userAgent string) (*models.Session, error) {
	for attempt := 0; attempt < 2; attempt++ {
		var s models.Session
		if err := db.First(&s, c.SessionID).Error; err != nil {
			return nil, ErrSessionInvalid
		}
		now := time.Now()
		if s.UserID != c.UserID || s.RevokedAt != nil || !now.Before(s.ExpiresAt) {
			return nil, ErrSessionInvalid
		}

		inGrace := s.PrevJTI != "" && c.JTI == s.PrevJTI &&
			s.RotatedAt != nil && now.Sub(*s.RotatedAt) <= GraceWindow
		if c.JTI != s.CurrentJTI && !inGrace {
			log.Printf("[AUTH] Refresh token dipakai ulang (sesi %d, user %d): sesi dicabut", s.ID, s.UserID)
			db.Model(&models.Session{}).
				Where("id = ? AND revoked_at IS NULL", s.ID).
				Update("revoked_at", now)
			return nil, ErrSessionInvalid
		}

		newJTI, err := utils.NewJTI()
		if err != nil {
			return nil, err
		}
		newExpiry := now.Add(utils.RefreshTokenTTL())

		// Update bersyarat (atomik): hanya berhasil bila sesi belum berubah sejak dibaca.
		res := db.Model(&models.Session{}).
			Where("id = ? AND current_jti = ? AND revoked_at IS NULL", s.ID, s.CurrentJTI).
			Updates(map[string]interface{}{
				"prev_jti":     s.CurrentJTI,
				"current_jti":  newJTI,
				"rotated_at":   now,
				"last_used_at": now,
				"expires_at":   newExpiry,
				"user_agent":   trimUA(userAgent),
			})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 1 {
			s.CurrentJTI = newJTI
			s.ExpiresAt = newExpiry
			return &s, nil
		}
		// Kalah balapan dengan refresh lain: baca ulang keadaan terbaru dan coba sekali lagi.
	}
	return nil, ErrSessionInvalid
}

// RevokeSession mencabut satu sesi (logout di satu perangkat).
func RevokeSession(db *gorm.DB, sessionID uint) error {
	return db.Model(&models.Session{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", time.Now()).Error
}

// RevokeAllSessions mencabut semua sesi milik user (logout semua perangkat).
func RevokeAllSessions(db *gorm.DB, userID uint) error {
	return db.Model(&models.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// IsSessionActive: sesi ada, belum dicabut, belum kedaluwarsa. Gagal baca DB = dianggap
// tidak aktif (fail closed).
func IsSessionActive(db *gorm.DB, sessionID uint) bool {
	var n int64
	err := db.Model(&models.Session{}).
		Where("id = ? AND revoked_at IS NULL AND expires_at > ?", sessionID, time.Now()).
		Count(&n).Error
	return err == nil && n > 0
}

// CleanupSessions menghapus sesi kedaluwarsa dan sesi yang dicabut lebih dari 30 hari lalu.
func CleanupSessions(db *gorm.DB) {
	now := time.Now()
	res := db.Where("expires_at < ? OR revoked_at < ?", now, now.Add(-30*24*time.Hour)).
		Delete(&models.Session{})
	if res.Error != nil {
		log.Println("[AUTH] Gagal membersihkan sesi:", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("[AUTH] %d sesi lama dibersihkan", res.RowsAffected)
	}
}

// StartSessionCleanup membersihkan sesi lama sekali saat start lalu setiap 6 jam.
func StartSessionCleanup(db *gorm.DB) {
	go func() {
		CleanupSessions(db)
		for range time.Tick(6 * time.Hour) {
			CleanupSessions(db)
		}
	}()
}
