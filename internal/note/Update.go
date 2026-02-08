package note

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	if err := h.svc.DB.Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "note not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	err = h.svc.DB.Transaction(func(tx *gorm.DB) error {
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

		if req.TagIDs != nil {
			var tags []models.Tag
			if len(*req.TagIDs) > 0 {
				if err := tx.Where("id IN ? AND user_id = ?", *req.TagIDs, userID).Find(&tags).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&note).Association("Tags").Replace(tags); err != nil {
				return err
			}
		}
		return tx.Preload("Tags").First(&note, note.ID).Error
	})
	if err != nil {
		zap.L().Error("Update note transaction failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}

	cacheKeyNote := "note:" + id
	cacheKeyAllNotes := fmt.Sprintf("notes:user:%d*", userID)

	_ = h.svc.Cache.Del(c, cacheKeyNote)
	_ = h.svc.Cache.ClearCacheByPattern(c, h.svc.Cache, cacheKeyAllNotes)

	zap.L().Info("Cache cleared for updated note", zap.String("note_id", id))

	go func(n models.Note) {
		textToEmbed := fmt.Sprintf("%s\n%s", n.Title, n.Content)

		vec, err := h.svc.AI.GetEmbedding(textToEmbed)
		if err != nil {
			zap.L().Error("Embedding generation failed", zap.Error(err))
			return
		}

		err = h.svc.Qdrant.Upsert(context.Background(), n.ID, vec, n.UserID, n.IsPrivate)
		if err != nil {
			zap.L().Error("Qdrant upsert failed", zap.Error(err))
		}
	}(note)

	utils.Success(c, note)
}
