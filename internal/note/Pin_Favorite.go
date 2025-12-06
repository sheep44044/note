package note

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"note/internal/cache"
	"note/internal/models"
	"note/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *NoteHandler) TogglePin(c *gin.Context) {
	id := c.Param("id")
	userid, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	userID, ok := userid.(uint)
	if !ok {
		utils.Error(c, http.StatusInternalServerError, "用户ID类型错误")
		return
	}

	var note models.Note
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "笔记不存在或无权操作")
		} else {
			utils.Error(c, http.StatusInternalServerError, "数据库错误")
		}
		return
	}

	// 切换状态
	note.IsPinned = !note.IsPinned

	if err := h.db.Save(&note).Error; err != nil {
		slog.Error("Toggle pin failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "操作失败")
		return
	}

	// 清理缓存（你已有的逻辑）
	cache.Del("note:" + id)
	cache.Del(fmt.Sprintf("notes:user:%d", userID))

	utils.Success(c, gin.H{
		"is_pinned": note.IsPinned,
	})
}

func (h *NoteHandler) ToggleFavorite(c *gin.Context) {
	id := c.Param("id")
	userid, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	userID, ok := userid.(uint)
	if !ok {
		utils.Error(c, http.StatusInternalServerError, "用户ID类型错误")
		return
	}

	var note models.Note
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "笔记不存在或无权操作")
		} else {
			utils.Error(c, http.StatusInternalServerError, "数据库错误")
		}
		return
	}

	note.IsFavorite = !note.IsFavorite

	if err := h.db.Save(&note).Error; err != nil {
		slog.Error("Toggle favorite failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "操作失败")
		return
	}

	cache.Del("note:" + id)
	cache.Del(fmt.Sprintf("notes:user:%d", userID))
	cache.Del(fmt.Sprintf("notes:favorites:%d", userID)) // 可选：单独缓存收藏列表

	utils.Success(c, gin.H{
		"is_favorite": note.IsFavorite,
	})
}

func (h *NoteHandler) ListFavorites(c *gin.Context) {
	userid, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	userID, ok := userid.(uint)
	if !ok {
		utils.Error(c, http.StatusInternalServerError, "用户ID类型错误")
		return
	}

	var notes []models.Note
	h.db.Where("user_id = ? AND is_favorite = ?", userID, true).
		Order("updated_at DESC").
		Find(&notes)

	utils.Success(c, notes)
}
