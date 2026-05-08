package models

import "time"

type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	AnimeID   string    `gorm:"type:varchar(100);not null;index" json:"anime_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	ParentID  *uint     `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Relationship
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies   []Comment `gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"replies,omitempty"`
}
