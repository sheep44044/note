package models

import "time"

type History struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"uniqueIndex:idx_user_note"`
	NoteID uint `gorm:"uniqueIndex:idx_user_note"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type HistoryMsg struct {
	UserID uint `json:"user_id"`
	NoteID uint `json:"note_id"`
}
