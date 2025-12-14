package note

import (
	"note/internal/cache"

	"gorm.io/gorm"
)

type NoteHandler struct {
	db    *gorm.DB
	cache *cache.RedisCache
}

func NewNoteHandler(db *gorm.DB, cache *cache.RedisCache) *NoteHandler {
	return &NoteHandler{db: db, cache: cache}
}
