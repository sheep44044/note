package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"note/internal/models"
	"note/internal/redis1"
	"note/internal/utils"
	"note/internal/validators"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NoteTag struct {
	db *gorm.DB
}

func NewNoteTag(db *gorm.DB) *NoteTag {
	return &NoteTag{db: db}
}

func (h *NoteTag) GetTags(c *gin.Context) {
	cacheKey := "tags:all"
	cachedTags, err := redis1.Get(cacheKey)
	if err == nil {
		var tags []models.Tag
		if err := json.Unmarshal([]byte(cachedTags), &tags); err == nil {
			slog.Debug("Tags retrieved from cache", "key", cacheKey)
			utils.Success(c, tags)
			return
		}
	}

	var tags []models.Tag
	h.db.Find(&tags)

	tagsJSON, _ := json.Marshal(tags)
	redis1.SetWithRandomTTL(cacheKey, string(tagsJSON), 10*time.Minute) // 10分钟TTL

	utils.Success(c, tags)
}

func (h *NoteTag) GetTag(c *gin.Context) {
	id := c.Param("id")
	cacheKey := "tag:" + id

	cachedTag, err := redis1.Get(cacheKey)
	if err == nil {
		var tag models.Tag
		if err := json.Unmarshal([]byte(cachedTag), &tag); err == nil {
			slog.Debug("Tags retrieved from cache", "key", cacheKey)
			utils.Success(c, tag)
			return
		}
	}

	var tag models.Tag
	if err := h.db.Where("id = ?", id).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "tag not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	tagJSON, _ := json.Marshal(tag)
	redis1.SetWithRandomTTL(cacheKey, string(tagJSON), 10*time.Minute)

	utils.Success(c, tag)
}

func (h *NoteTag) CreateTag(c *gin.Context) {
	var req validators.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "invalid tag")
		return
	}

	tag := models.Tag{
		Name:  req.Name,
		Color: req.Color,
	}
	h.db.Create(&tag)

	cacheKeyAllTags := "Tags:all"
	redis1.Del(cacheKeyAllTags)

	utils.Success(c, tag)
}

func (h *NoteTag) UpdateTag(c *gin.Context) {
	id := c.Param("id")
	var req validators.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var tag models.Tag
	if err := h.db.Where("id = ?", id).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "tag not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}
	h.db.Model(&tag).Updates(models.Tag{
		Name:  req.Name,
		Color: req.Color,
	})

	// 更新成功后，清理相关缓存
	cacheKeyTag := "tag:" + id
	cacheKeyAllTags := "tags:all"

	redis1.Del(cacheKeyTag)
	redis1.Del(cacheKeyAllTags)
	slog.Info("Cache cleared for updated note", "note_id", id)

	utils.Success(c, tag)
}

func (h *NoteTag) DeleteTag(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		utils.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	result := h.db.Delete(&models.Tag{}, id)
	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, "tag not found")
		return
	}

	cacheKeyTag := "tag:" + c.Param("id")
	cacheKeyAllTags := "tags:all"

	redis1.Del(cacheKeyTag)
	redis1.Del(cacheKeyAllTags)

	slog.Info("Tag and related caches cleared", "tag_id", id)
	utils.Success(c, gin.H{"message": "deleted"})
}
