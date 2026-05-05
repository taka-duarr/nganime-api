package models

import "time"

type User struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	Password  string     `gorm:"not null" json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Bookmarks []Bookmark `gorm:"foreignKey:UserID" json:"bookmarks,omitempty"`
}
