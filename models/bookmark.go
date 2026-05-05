package models

import "time"

type Bookmark struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	AnimeID   string    `gorm:"type:varchar(100);not null;index" json:"anime_id"`
	Title     string    `gorm:"not null" json:"title"`
	Poster    string    `json:"poster"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
