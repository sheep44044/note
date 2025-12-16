package models

import "time"

type Reaction struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"index"`
	NoteID uint   `gorm:"index"`
	Emoji  string `gorm:"size:10"` // 存 "❤️", "👍", "🔥" 等

	CreatedAt time.Time
}

type ReactionMsg struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"index"`
	NoteID uint   `gorm:"index"`
	Emoji  string `gorm:"size:10"`
	Action string `json:"action"` // "add" or "remove"
}

// 防止同一用户对同一笔记重复点同一个 emoji

func (Reaction) TableName() string {
	return "reactions"
}
