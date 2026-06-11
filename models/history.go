package models

import "time"

type WatchHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index:idx_user_anime_episode,unique" json:"user_id"`
	AnimeID   string    `gorm:"type:varchar(100);not null;index:idx_user_anime_episode,unique" json:"anime_id"`
	EpisodeID string    `gorm:"type:varchar(100);not null;index:idx_user_anime_episode,unique" json:"episode_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
