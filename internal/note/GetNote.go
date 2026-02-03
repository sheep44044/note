package note

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (h *NoteHandler) GetNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	tagIDStr := c.Query("tag_id")

	cacheKey := fmt.Sprintf("notes:user:%d", userID)
	if tagIDStr != "" {
		cacheKey = fmt.Sprintf("notes:user:%d:tag:%s", userID, tagIDStr)
	}

	cachedNotes, err := h.cache.Get(c, cacheKey)
	if err == nil {
		var notes []models.Note
		if err := json.Unmarshal([]byte(cachedNotes), &notes); err == nil {
			zap.L().Debug("Notes retrieved from cache", zap.String("key", cacheKey))
			utils.Success(c, notes)
			return
		}
	}

	var notes []models.Note
	query := h.db.Model(&models.Note{}).Where("user_id = ?", userID)

	if tagIDStr != "" {
		query = query.Joins("JOIN note_tags ON note_tags.note_id = notes.id").
			Where("note_tags.tag_id = ?", tagIDStr)
	}

	err = query.Preload("Tags").
		Order("is_pinned DESC").
		Order("updated_at DESC").
		Find(&notes).Error

	if err != nil {
		zap.L().Error("db query notes failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	notesJSON, _ := json.Marshal(notes)
	_ = h.cache.SetWithRandomTTL(c, cacheKey, string(notesJSON), 10*time.Minute)

	utils.Success(c, notes)
}

func (h *NoteHandler) GetNote(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	id := c.Param("id")
	noteID, _ := strconv.ParseUint(id, 10, 64)
	cacheKey := "note:" + id

	cachedNote, err := h.cache.Get(c, cacheKey)
	if err == nil {
		var note models.Note
		if err := json.Unmarshal([]byte(cachedNote), &note); err == nil {
			zap.L().Debug("Note retrieved from cache", zap.String("key", cacheKey))

			h.recordNoteView(c, userID, uint(noteID))

			utils.Success(c, note)
			return
		}
	}

	var note models.Note
	if err := h.db.Preload("Tags").Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "note not found")
		} else {
			zap.L().Error("db query note failed", zap.String("note_id", id), zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	noteJSON, _ := json.Marshal(note)
	_ = h.cache.SetWithRandomTTL(c, cacheKey, string(noteJSON), 10*time.Minute)

	h.recordNoteView(c, userID, uint(noteID))

	utils.Success(c, note)
}

func (h *NoteHandler) GetFollowingFeed(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 20
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	key := fmt.Sprintf("timeline:user:%d", userID)

	noteIDStrs, err := h.cache.LRange(c, key, start, stop)

	if len(noteIDStrs) == 0 {
		if page > 1 {
			utils.Success(c, []models.Note{})
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
			utils.Success(c, []models.Note{})
			return
		}

		var notes []models.Note
		h.db.Preload("Tags").
			Where("user_id IN ?", followedIDs).
			Where("is_private = ?", false).
			Order("created_at DESC").
			Offset(int(start)).
			Limit(pageSize).
			Find(&notes)

		utils.Success(c, notes)
		return
	}

	var notes []models.Note
	err = h.db.Preload("Tags").
		Where("id IN ?", noteIDStrs).
		Where("is_private = ?", false).
		Find(&notes).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	noteMap := make(map[uint]models.Note)
	for _, n := range notes {
		noteMap[n.ID] = n
	}

	var sortedNotes []models.Note
	for _, idStr := range noteIDStrs {
		id, _ := strconv.Atoi(idStr)
		if n, ok := noteMap[uint(id)]; ok {
			sortedNotes = append(sortedNotes, n)
		}
	}

	utils.Success(c, sortedNotes)
}
