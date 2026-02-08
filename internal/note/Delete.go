package note

import (
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *NoteHandler) DeleteNote(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		utils.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	result := h.svc.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Note{})
	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, "note not found")
		return
	}

	cacheKeyNote := "note:" + c.Param("id")
	cacheKeyAllNotes := fmt.Sprintf("notes:user:%d*", userID)

	_ = h.svc.Cache.Del(c, cacheKeyNote)
	_ = h.svc.Cache.ClearCacheByPattern(c, h.svc.Cache, cacheKeyAllNotes)

	zap.L().Info("Cache cleared for deleted note", zap.Int("note_id", id))
	utils.Success(c, gin.H{"message": "deleted"})
}
