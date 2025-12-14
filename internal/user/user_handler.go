package user

import (
	"note/config"
	"note/internal/cache"

	"gorm.io/gorm"
)

type UserHandler struct {
	db    *gorm.DB
	cache *cache.RedisCache
	cfg   *config.Config
}

func NewUserHandler(db *gorm.DB, cfg *config.Config, cache *cache.RedisCache) *UserHandler {
	return &UserHandler{db: db, cfg: cfg, cache: cache}
}
