package models

import "time"

type Favorite struct {
	UserID uint `gorm:"primaryKey"`
	NoteID uint `gorm:"primaryKey"`

	CreatedAt time.Time
}

type FavoriteMsg struct {
	UserID uint   `json:"user_id"`
	NoteID uint   `json:"note_id"`
	Action string `json:"action"` // "add" or "remove"
}
