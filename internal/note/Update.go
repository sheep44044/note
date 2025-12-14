package note

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *NoteHandler) UpdateNote(c *gin.Context) {
	id := c.Param("id")
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

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

	err = h.db.Transaction(func(tx *gorm.DB) error {
		update := make(map[string]interface{})
		if req.Title != nil {
			update["title"] = *req.Title
		}
		if req.Content != nil {
			update["content"] = *req.Content
		}
		if req.IsPrivate != nil {
			update["is_private"] = *req.IsPrivate
		}
		if len(update) > 0 {
			if err := tx.Model(&note).Updates(update).Error; err != nil {
				return err
			}
		}

		if len(req.TagIDs) > 0 {
			var tags []models.Tag
			if err := tx.Where("id IN ? AND user_id IN ?", req.TagIDs, userID).Find(&tags).Error; err != nil {
				return err
			}
			result := tx.Model(&note).Association("Tags").Replace(tags)
			if result != nil {
				return result
			}
		}
		return tx.Preload("Tags").First(&note, note.ID).Error
	})
	if err != nil {
		slog.Error("Update note transaction failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}

	cacheKeyNote := "note:" + id
	cacheKeyAllNotes := fmt.Sprintf("notes:user:%d", userID)

	h.cache.Del(c, cacheKeyNote)
	h.cache.Del(c, cacheKeyAllNotes)
	slog.Info("Cache cleared for updated note", "note_id", id)

	utils.Success(c, note)
}
