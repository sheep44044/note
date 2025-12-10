package note

import (
	"fmt"
	"net/http"
	"note/internal/cache"
	"note/internal/models"
	"note/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// GetRecentNotes 返回最近访问的笔记ID列表（最多5个，按时间倒序）
func (h *NoteHandler) GetRecentNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	key := fmt.Sprintf("user:history:%d", userID)

	noteIDs, err := cache.ZRevRange(key, 0, 4)
	if err != nil {
		// Redis 出错或 key 不存在都返回空列表（更友好）
		noteIDs = []string{}
	}

	if len(noteIDs) == 0 {
		utils.Success(c, []interface{}{}) // 返回空数组 []
		return
	}

	var notes []models.Note
	if err := h.db.Where("id = ?", noteIDs).Find(&notes).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
	}

	noteMap := make(map[uint]models.Note)
	for _, n := range notes {
		noteMap[n.ID] = n
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
				IsFavorite:    note.IsFavorite,
				CreatedAt:     note.CreatedAt,
			})
		}
	}
	utils.Success(c, result)
}

// recordNoteView 记录用户访问某篇笔记（内部调用，小写开头）
func (h *NoteHandler) recordNoteView(userID, noteID string) {
	key := "user:history:" + userID
	now := float64(time.Now().Unix())

	// 1. 先移除旧记录（实现去重）
	cache.ZRem(key, noteID)

	// 2. 添加新记录（以当前时间戳为分数）
	cache.ZAdd(key, redis.Z{Score: now, Member: noteID})

	// 3. 只保留最近5条（-6 表示从第0名到倒数第6名，共删掉超出的部分）
	cache.ZRemRangeByRank(key, 0, -6)

	// 4. 设置30天自动过期（可选但推荐）
	cache.Expire(key, 30*24*time.Hour)
}
