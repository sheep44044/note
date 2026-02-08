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

func (h *NoteHandler) TogglePin(c *gin.Context) {
	id := c.Param("id")
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	var note models.Note
	if err := h.svc.DB.Select("id, is_pinned, user_id").Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "笔记不存在或无权操作")
		} else {
			utils.Error(c, http.StatusInternalServerError, "数据库错误")
		}
		return
	}

	newValue := !note.IsPinned

	if err := h.svc.DB.Model(&note).Update("is_pinned", newValue).Error; err != nil {
		zap.L().Error("Toggle pin failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "操作失败")
		return
	}

	_ = h.svc.Cache.Del(c, "note:"+id)
	_ = h.svc.Cache.Del(c, fmt.Sprintf("notes:user:%d", userID))

	message := "已取消置顶"
	if newValue {
		message = "置顶成功"
	}

	utils.Success(c, gin.H{
		"is_pinned": newValue,
		"message":   message,
	})
}

func (h *NoteHandler) FavoriteNote(c *gin.Context) {
	noteID := c.Param("id")
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	var note models.Note
	if err := h.svc.DB.Select("id, is_private, favorite_count").Where("id = ? AND is_private = ?", noteID, false).First(&note).Error; err != nil {
		utils.Error(c, http.StatusNotFound, "笔记不存在或不可公开访问")
		return
	}

	msg := models.FavoriteMsg{UserID: userID, NoteID: note.ID, Action: "add"}
	body, err := json.Marshal(msg)
	if err != nil {
		zap.L().Error("JSON marshal failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "系统错误")
		return
	}

	if err := h.svc.Rabbit.Publish("favorite_queue", body); err != nil {
		zap.L().Error("MQ publish failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "收藏失败，请稍后重试")
		return
	}

	_ = h.svc.Cache.Del(c, "note:"+noteID)
	_ = h.svc.Cache.Del(c, fmt.Sprintf("notes:favorites:%d", userID))

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
	body, err := json.Marshal(msg)
	if err != nil {
		zap.L().Error("JSON marshal failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "系统错误")
		return
	}

	if err := h.svc.Rabbit.Publish("favorite_queue", body); err != nil {
		zap.L().Error("MQ publish failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "取消收藏失败，请稍后重试")
		return
	}

	_ = h.svc.Cache.Del(c, "note:"+noteIDstr)
	_ = h.svc.Cache.Del(c, fmt.Sprintf("notes:favorites:%d", userID))

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
	h.svc.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&favorites)

	if len(favorites) == 0 {
		utils.Success(c, []interface{}{})
		return
	}

	noteIDs := make([]uint, len(favorites))
	for i, f := range favorites {
		noteIDs[i] = f.NoteID
	}

	var notes []models.Note
	h.svc.DB.Preload("Tags").
		Where("id IN ?", noteIDs).
		Where("is_private = ?", false).
		Find(&notes)

	noteMap := make(map[uint]models.Note)
	for _, n := range notes {
		noteMap[n.ID] = n
	}

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
