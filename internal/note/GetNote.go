package note

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"note/internal/cache"
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
	cachedNotes, err := cache.Get(cacheKey)
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
	cache.SetWithRandomTTL(cacheKey, string(notesJSON), 10*time.Minute)

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

	cachedNote, err := cache.Get(cacheKey)
	if err == nil {
		var note models.Note
		if err := json.Unmarshal([]byte(cachedNote), &note); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)

			h.recordNoteView(strconv.Itoa(int(userID)), id)

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
	cache.SetWithRandomTTL(cacheKey, string(noteJSON), 10*time.Minute)

	h.recordNoteView(strconv.Itoa(int(userID)), id)

	utils.Success(c, note)
}
