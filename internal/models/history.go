package models

import "time"

type History struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index"` // 加索引方便查询
	NoteID    uint      `gorm:"index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type HistoryMsg struct {
	UserID uint `json:"user_id"`
	NoteID uint `json:"note_id"`
}
