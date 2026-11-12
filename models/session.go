package models

import "time"

// Session mewakili satu login (satu perangkat/browser).
//
// Refresh token dirotasi setiap dipakai: hanya token dengan JTI terbaru (CurrentJTI)
// yang sah. Memakai ulang token lama di luar masa tenggang dianggap pencurian token
// dan seluruh sesi dicabut. Access token membawa ID sesi ini, jadi mencabut sesi
// (logout, logout semua perangkat) langsung mematikan access token-nya juga.
//
// Yang disimpan hanya ID acak (JTI), bukan token. JTI saja tidak bisa dipakai untuk
// memalsukan token karena token tetap harus ditandatangani dengan JWT_SECRET.
type Session struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	CurrentJTI string     `gorm:"type:varchar(64);not null" json:"-"`
	PrevJTI    string     `gorm:"type:varchar(64)" json:"-"`
	RotatedAt  *time.Time `json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt  *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	LastUsedAt time.Time  `json:"last_used_at"`
	UserAgent  string     `gorm:"type:varchar(255)" json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
}
