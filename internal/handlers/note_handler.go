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

type NoteHandler struct {
	db *gorm.DB
}

func NewNoteHandler(db *gorm.DB) *NoteHandler {
	return &NoteHandler{db: db}
}

func (h *NoteHandler) GetNotes(c *gin.Context) {
	// 1. 先尝试从缓存获取
	cacheKey := "notes:all"
	cachedNotes, err := redis1.Get(cacheKey)
	if err == nil {
		var notes []models.Note
		if err := json.Unmarshal([]byte(cachedNotes), &notes); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)
			utils.Success(c, notes)
			return
		}
	}

	var notes []models.Note
	if err := h.db.Preload("Tags").Find(&notes).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	// 3. 将结果存入缓存
	notesJSON, _ := json.Marshal(notes)
	redis1.SetWithRandomTTL(cacheKey, string(notesJSON), 10*time.Minute) // 10分钟TTL

	utils.Success(c, notes)
}

func (h *NoteHandler) GetNote(c *gin.Context) {
	id := c.Param("id")
	cacheKey := "note:" + id

	cachedNote, err := redis1.Get(cacheKey)
	if err == nil {
		var note models.Note
		if err := json.Unmarshal([]byte(cachedNote), &note); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)
			utils.Success(c, note)
			return
		}
	}

	var note models.Note
	if err := h.db.Preload("Tags").Where("id = ?", id).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "note not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	noteJSON, _ := json.Marshal(note)
	redis1.SetWithRandomTTL(cacheKey, string(noteJSON), 10*time.Minute)

	utils.Success(c, note)
}

func (h *NoteHandler) CreateNote(c *gin.Context) {
	var req validators.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "invalid note")
		return
	}

	var tags []models.Tag
	if len(req.TagIDs) > 0 {
		h.db.Where("id IN ?", req.TagIDs).Find(&tags)
	}

	note := models.Note{
		Title:   req.Title,
		Content: req.Content,
		Tags:    tags,
	}

	h.db.Create(&note)

	cacheKeyAllNotes := "notes:all"
	redis1.Del(cacheKeyAllNotes)

	utils.Success(c, note)
}

func (h *NoteHandler) UpdateNote(c *gin.Context) {
	id := c.Param("id")

	var req validators.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var note models.Note
	if err := h.db.First(&note, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "note not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	h.db.Model(&note).Updates(models.Note{
		Title:   req.Title,
		Content: req.Content,
	})

	var tags []models.Tag
	if len(req.TagIDs) > 0 {
		h.db.Where("id IN ?", req.TagIDs).Find(&tags)
	}
	h.db.Model(&note).Association("Tags").Replace(tags)

	h.db.Preload("Tags").First(&note, note.ID)

	// 更新成功后，清理相关缓存
	cacheKeyNote := "note:" + id
	cacheKeyAllNotes := "notes:all"

	redis1.Del(cacheKeyNote)
	redis1.Del(cacheKeyAllNotes)
	slog.Info("Cache cleared for updated note", "note_id", id)

	utils.Success(c, note)
}

func (h *NoteHandler) DeleteNote(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		utils.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	result := h.db.Delete(&models.Note{}, id)
	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, "note not found")
		return
	}

	cacheKeyNote := "note:" + c.Param("id")
	cacheKeyAllNotes := "notes:all"

	redis1.Del(cacheKeyNote)
	redis1.Del(cacheKeyAllNotes)

	slog.Info("Cache cleared for deleted note", "note_id", id)
	utils.Success(c, gin.H{"message": "deleted"})
}
