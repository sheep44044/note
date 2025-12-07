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

func (h *NoteHandler) FavoriteNote(c *gin.Context) {
	noteID := c.Param("id")
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

	// 确保笔记是 public
	var note models.Note
	if err := h.db.Select("id, is_private, favorite_count").Where("id = ? AND is_private = ?", noteID, false).First(&note).Error; err != nil {
		utils.Error(c, http.StatusNotFound, "笔记不存在或不可公开访问")
		return
	}

	// 创建收藏记录
	fav := models.Favorite{UserID: userID, NoteID: note.ID}
	if err := h.db.Create(&fav).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			utils.Success(c, gin.H{"message": "已收藏"})
			return
		}
		utils.Error(c, http.StatusInternalServerError, "操作失败")
		return
	}

	// 更新计数（+1）
	h.db.Model(&note).Update("favorite_count", gorm.Expr("favorite_count + 1"))

	// 清缓存
	cache.Del("note:" + noteID)
	cache.Del(fmt.Sprintf("notes:favorites:%d", userID))

	utils.Success(c, gin.H{"favorite_count": note.FavoriteCount + 1})
}

func (h *NoteHandler) UnfavoriteNote(c *gin.Context) {
	noteID := c.Param("id")
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

	h.db.Where("user_id = ? AND note_id = ?", userID, noteID).Delete(&models.Favorite{})

	// 更新计数（-1）
	h.db.Model(&models.Note{}).Where("id = ?", noteID).Update("favorite_count", gorm.Expr("GREATEST(favorite_count - 1, 0)"))

	cache.Del("note:" + noteID)
	cache.Del(fmt.Sprintf("notes:favorites:%d", userID))

	utils.Success(c, gin.H{"message": "已取消收藏"})
}
