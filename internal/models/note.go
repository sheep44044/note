package models

import (
	"time"
)

type Note struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	UserID  uint   `gorm:"index"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`

	IsPrivate     bool `gorm:"default:false" json:"is_private"`
	IsPinned      bool `gorm:"default:false;index"` // 是否置顶
	IsFavorite    bool `gorm:"default:false;index"`
	FavoriteCount int  `gorm:"default:0"` // 收藏总数

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Tags      []Tag     `gorm:"many2many:note_tags;"`
}

type FeedMsg struct {
	AuthorID uint  `json:"author_id"`
	NoteID   uint  `json:"note_id"`
	PostTime int64 `json:"post_time"`
}
