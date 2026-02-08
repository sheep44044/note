package note

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// GetRecentNotes 返回最近访问的笔记ID列表（最多5个，按时间倒序）
func (h *NoteHandler) GetRecentNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	key := fmt.Sprintf("user:history:%d", userID)

	noteIDs, err := h.svc.Cache.ZRevRange(c, key, 0, 4)
	if err != nil {
		// Redis 出错或 key 不存在都返回空列表（更友好）
		noteIDs = []string{}
	}

	if len(noteIDs) == 0 {
		var histories []models.History
		if err := h.svc.DB.Where("user_id = ?", userID).Order("updated_at DESC").Limit(5).Find(&histories).Error; err == nil {
			if len(histories) > 0 {
				ctx := context.Background()
				pipe := h.svc.Cache.Pipeline()
				for _, h := range histories {
					noteIDStr := strconv.Itoa(int(h.NoteID))
					noteIDs = append(noteIDs, noteIDStr)
					pipe.ZAdd(ctx, key, redis.Z{Score: float64(h.UpdatedAt.Unix()), Member: noteIDStr})
				}
				pipe.Expire(ctx, key, 30*24*time.Hour)
				if _, err := pipe.Exec(ctx); err != nil {
					zap.L().Warn("failed to warm up history cache", zap.Uint("user_id", userID), zap.Error(err))
				}
			}
		}
	}

	if len(noteIDs) == 0 {
		utils.Success(c, []interface{}{})
		return
	}

	var notes []models.Note
	err = h.svc.DB.Where("id IN ?", noteIDs).
		Where(h.svc.DB.Where("is_private = ?", false).
			Or("user_id = ?", userID)).
		Find(&notes).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	noteMap := make(map[uint]models.Note)
	noteIDsUint := make([]uint, 0, len(notes))
	for _, n := range notes {
		noteMap[n.ID] = n
		noteIDsUint = append(noteIDsUint, n.ID)
	}

	favoriteMap := make(map[uint]bool)
	if len(noteIDsUint) > 0 {
		var favorites []models.Favorite
		h.svc.DB.Where("user_id = ? AND note_id IN ?", userID, noteIDsUint).Find(&favorites)
		for _, f := range favorites {
			favoriteMap[f.NoteID] = true
		}
	}

	type NoteDTO struct {
		ID            uint      `json:"id"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		FavoriteCount int       `json:"favorite_count"`
		IsFavorite    bool      `json:"is_favorite"`
		CreatedAt     time.Time `json:"created_at"`
	}
	result := make([]NoteDTO, 0, len(noteIDs))

	for _, idStr := range noteIDs {
		idUint64, _ := strconv.ParseUint(idStr, 10, 32)
		idUint := uint(idUint64)

		if note, exists := noteMap[idUint]; exists {
			result = append(result, NoteDTO{
				ID:            note.ID,
				Title:         note.Title,
				Content:       note.Content,
				FavoriteCount: note.FavoriteCount,
				IsFavorite:    favoriteMap[note.ID],
				CreatedAt:     note.CreatedAt,
			})
		}
	}
	utils.Success(c, result)
}

func (h *NoteHandler) recordNoteView(ctx context.Context, userID, noteID uint) {
	key := fmt.Sprintf("user:history:%d", userID)
	now := float64(time.Now().Unix())

	noteIDStr := strconv.Itoa(int(noteID))

	pipe := h.svc.Cache.Pipeline()

	pipe.ZRem(ctx, key, noteIDStr)
	pipe.ZAdd(ctx, key, redis.Z{Score: now, Member: noteIDStr})
	pipe.ZRemRangeByRank(ctx, key, 0, -6)
	pipe.Expire(ctx, key, 30*24*time.Hour)

	if _, err := pipe.Exec(ctx); err != nil {
		zap.L().Error("failed to update note view history in redis", zap.Uint("user_id", userID), zap.Uint("note_id", noteID), zap.Error(err))
	}

	if h.svc.Rabbit != nil {
		msg := models.HistoryMsg{UserID: userID, NoteID: noteID}
		body, err := json.Marshal(msg)

		if err != nil {
			zap.L().Error("failed to marshal history msg", zap.Error(err))
			return
		}
		if err := h.svc.Rabbit.Publish("history_queue", body); err != nil {
			zap.L().Error("failed to publish history msg to rabbitmq", zap.Uint("user_id", userID), zap.Error(err))
		}
	} else {
		zap.L().Warn("rabbitmq is nil, skipping history publish", zap.Uint("user_id", userID))
	}
}
