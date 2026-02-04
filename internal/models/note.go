package models

import (
	"time"
)

type Note struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	UserID  uint   `gorm:"index"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`

	IsPrivate      bool           `gorm:"default:false" json:"is_private"`
	IsPinned       bool           `gorm:"default:false;index"`
	IsFavorite     bool           `gorm:"default:false;index"`
	FavoriteCount  int            `gorm:"default:0"`
	ReactionCounts map[string]int `json:"reaction_counts" gorm:"serializer:json;default:{}"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Tags      []Tag     `gorm:"many2many:note_tags;"`
	Summary   string    `json:"summary" gorm:"type:text"`
}

type FeedMsg struct {
	AuthorID uint  `json:"author_id"`
	NoteID   uint  `json:"note_id"`
	PostTime int64 `json:"post_time"`
}

// 新增：AI 任务消息结构
type AITaskMsg struct {
	NoteID uint   `json:"note_id"`
	Task   string `json:"task"` // "generate_title" 或 "generate_summary"
}
