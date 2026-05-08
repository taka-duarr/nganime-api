package models

import "time"

type User struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	Password  string     `gorm:"not null" json:"-"`
	ProfilePicture string     `json:"profile_picture"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Bookmarks []Bookmark `gorm:"foreignKey:UserID" json:"bookmarks,omitempty"`
	Comments  []Comment  `gorm:"foreignKey:UserID" json:"comments,omitempty"`
}
