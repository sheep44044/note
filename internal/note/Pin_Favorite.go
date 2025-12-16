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

func (h *NoteHandler) TogglePin(c *gin.Context) {
	id := c.Param("id")
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
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
	h.cache.Del(c, "note:"+id)
	h.cache.Del(c, fmt.Sprintf("notes:user:%d", userID))

	utils.Success(c, gin.H{
		"is_pinned": note.IsPinned,
	})
}

func (h *NoteHandler) FavoriteNote(c *gin.Context) {
	noteID := c.Param("id")
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 确保笔记是 public
	var note models.Note
	if err := h.db.Select("id, is_private, favorite_count").Where("id = ? AND is_private = ?", noteID, false).First(&note).Error; err != nil {
		utils.Error(c, http.StatusNotFound, "笔记不存在或不可公开访问")
		return
	}

	msg := models.FavoriteMsg{UserID: userID, NoteID: note.ID, Action: "add"}
	body, _ := json.Marshal(msg)
	h.rabbit.Publish("favorite_queue", body)

	// 清缓存
	h.cache.Del(c, "note:"+noteID)
	h.cache.Del(c, fmt.Sprintf("notes:favorites:%d", userID))

	utils.Success(c, gin.H{"favorite_count": note.FavoriteCount + 1})
}

func (h *NoteHandler) UnfavoriteNote(c *gin.Context) {
	noteIDstr := c.Param("id")
	noteID, _ := strconv.ParseUint(noteIDstr, 0, 64)

	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	msg := models.FavoriteMsg{UserID: userID, NoteID: uint(noteID), Action: "remove"}
	body, _ := json.Marshal(msg)
	h.rabbit.Publish("favorite_queue", body)

	h.cache.Del(c, "note:"+noteIDstr)
	h.cache.Del(c, fmt.Sprintf("notes:favorites:%d", userID))

	utils.Success(c, gin.H{"message": "已取消收藏"})
}

func (h *NoteHandler) ListMyFavorites(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 20

	var favorites []models.Favorite
	h.db.Preload("Note").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset((page - 1) * limit).
		Find(&favorites)

	noteIDs := make([]uint, len(favorites))
	for i, f := range favorites {
		noteIDs[i] = f.NoteID
	}

	var notes []models.Note
	h.db.Where("id IN ?", noteIDs).Find(&notes)
	noteMap := make(map[uint]models.Note)
	for _, n := range notes {
		noteMap[n.ID] = n
	}

	// 构造结果（按收藏时间倒序）
	type FavNote struct {
		Note    models.Note `json:"note"`
		FavedAt time.Time   `json:"faved_at"`
	}
	result := make([]FavNote, 0, len(favorites))
	for _, f := range favorites {
		if note, exists := noteMap[f.NoteID]; exists {
			result = append(result, FavNote{Note: note, FavedAt: f.CreatedAt})
		}
	}

	utils.Success(c, result)
}
