package note

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *NoteHandler) GetNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 1. 先尝试从缓存获取
	cacheKey := fmt.Sprintf("notes:user:%d", userID)
	cachedNotes, err := h.cache.Get(c, cacheKey)
	if err == nil {
		var notes []models.Note
		if err := json.Unmarshal([]byte(cachedNotes), &notes); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)
			utils.Success(c, notes)
			return
		}
	}

	var notes []models.Note
	err = h.db.Preload("Tags").
		Where("user_id = ?", userID).
		Order("is_pinned DESC").
		Order("updated_at DESC").
		Find(&notes).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	// 3. 将结果存入缓存
	notesJSON, _ := json.Marshal(notes)
	h.cache.SetWithRandomTTL(c, cacheKey, string(notesJSON), 10*time.Minute)

	utils.Success(c, notes)
}

func (h *NoteHandler) GetNote(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	id := c.Param("id")
	cacheKey := "note:" + id

	cachedNote, err := h.cache.Get(c, cacheKey)
	if err == nil {
		var note models.Note
		if err := json.Unmarshal([]byte(cachedNote), &note); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)

			h.recordNoteView(c, strconv.Itoa(int(userID)), id)

			utils.Success(c, note)
			return
		}
	}

	var note models.Note
	if err := h.db.Preload("Tags").Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "note not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	noteJSON, _ := json.Marshal(note)
	h.cache.SetWithRandomTTL(c, cacheKey, string(noteJSON), 10*time.Minute)

	h.recordNoteView(c, strconv.Itoa(int(userID)), id)

	utils.Success(c, note)
}

func (h *NoteHandler) GetFollowingFeed(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	var followedIDs []uint
	err = h.db.Model(&models.UserFollow{}).
		Where("follower_id = ?", userID).
		Pluck("followed_id", &followedIDs).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "database error")
	}

	if len(followedIDs) == 0 {
		utils.Success(c, []models.Note{}) // 没关注任何人，返回空
		return
	}

	var notes []models.Note
	h.db.Preload("Tags").
		Where("user_id IN ?", followedIDs).
		Where("is_private = ?", false).
		Order("created_at DESC").
		Limit(20).
		Find(&notes)

	utils.Success(c, notes)
}
