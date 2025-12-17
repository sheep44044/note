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

func (h *NoteHandler) GetNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 1. 先尝试从缓存获取
	cacheKey := fmt.Sprintf("notes:user:%d", userID)
	cachedNotes, err := h.cache.Get(c, cacheKey)
	if err == nil {
		var notes []models.Note
		if err := json.Unmarshal([]byte(cachedNotes), &notes); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)
			utils.Success(c, notes)
			return
		}
	}

	var notes []models.Note
	err = h.db.Preload("Tags").
		Where("user_id = ?", userID).
		Order("is_pinned DESC").
		Order("updated_at DESC").
		Find(&notes).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	// 3. 将结果存入缓存
	notesJSON, _ := json.Marshal(notes)
	h.cache.SetWithRandomTTL(c, cacheKey, string(notesJSON), 10*time.Minute)

	utils.Success(c, notes)
}

func (h *NoteHandler) GetNote(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	id := c.Param("id")
	noteID, _ := strconv.ParseUint(id, 0, 64)
	cacheKey := "note:" + id

	cachedNote, err := h.cache.Get(c, cacheKey)
	if err == nil {
		var note models.Note
		if err := json.Unmarshal([]byte(cachedNote), &note); err == nil {
			slog.Debug("Notes retrieved from cache", "key", cacheKey)

			h.recordNoteView(c, userID, uint(noteID))

			utils.Success(c, note)
			return
		}
	}

	var note models.Note
	if err := h.db.Preload("Tags").Where("id = ? AND user_id = ?", id, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "note not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	noteJSON, _ := json.Marshal(note)
	h.cache.SetWithRandomTTL(c, cacheKey, string(noteJSON), 10*time.Minute)

	h.recordNoteView(c, userID, uint(noteID))

	utils.Success(c, note)
}

func (h *NoteHandler) GetFollowingFeed(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// [新增] 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	pageSize := 20
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	// 1. 直接从 Redis 个人收件箱读取 ID 列表
	key := fmt.Sprintf("timeline:user:%d", userID)

	noteIDStrs, err := h.cache.LRange(c, key, start, stop)

	// 2. 降级策略 (Cold Start)：如果 Redis 里是空的（比如刚注册，或者 Redis 数据丢了）
	// 仍然可以回退到原来的 SQL 逻辑，或者直接返回空让用户去"关注"一些人
	if len(noteIDStrs) == 0 {
		var followedIDs []uint
		err = h.db.Model(&models.UserFollow{}).
			Where("follower_id = ?", userID).
			Pluck("followed_id", &followedIDs).Error

		if err != nil {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}

		if len(followedIDs) == 0 {
			utils.Success(c, []models.Note{}) // 没关注任何人，返回空
			return
		}

		var notes []models.Note
		h.db.Preload("Tags").
			Where("user_id IN ?", followedIDs).
			Where("is_private = ?", false).
			Order("created_at DESC").
			Offset(int(start)). // [分页] 跳过前面页的数据
			Limit(pageSize).    // [分页] 只取当前页
			Find(&notes)

		utils.Success(c, notes)
		return
	}

	if len(noteIDStrs) == 0 {
		// Redis 有 key 但范围越界（比如只有 10 条，你查第 2 页），直接返回空
		utils.Success(c, []models.Note{})
		return
	}

	// 3. 根据 ID 列表去 MySQL 批量拉取详情 (WHERE id IN (...))
	// 注意：MySQL 返回的顺序可能不按 ID 顺序，需要自己在内存里重排一下
	var notes []models.Note
	h.db.Preload("Tags").
		Where("id IN ?", noteIDStrs).
		Where("is_private = ?", false).
		Find(&notes)

	// 4. 按 Redis 中的 ID 顺序重排 notes (保持时间倒序)
	noteMap := make(map[uint]models.Note)
	for _, n := range notes {
		noteMap[n.ID] = n
	}

	var sortedNotes []models.Note
	for _, idStr := range noteIDStrs {
		id, _ := strconv.Atoi(idStr)
		if n, ok := noteMap[uint(id)]; ok {
			sortedNotes = append(sortedNotes, n)
		}
	}

	utils.Success(c, sortedNotes)
}
