package note

import (
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *NoteHandler) ListPublicNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	sortBy := c.DefaultQuery("sort", "time") // time / popular
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 20

	query := h.db.Where("is_private = ?", false)

	switch sortBy {
	case "popular":
		query = query.Order("favorite_count DESC, created_at DESC")
	default: // time
		query = query.Order("created_at DESC")
	}

	var notes []models.Note
	h.db.Limit(limit).Offset((page - 1) * limit).Find(&notes)

	// 注入“当前用户是否已收藏”
	noteIDs := make([]uint, len(notes))
	for i, n := range notes {
		noteIDs[i] = n.ID
	}

	var favorites []models.Favorite
	h.db.Where("user_id = ? AND note_id IN ?", userID, noteIDs).Find(&favorites)
	favSet := make(map[uint]bool)
	for _, f := range favorites {
		favSet[f.NoteID] = true
	}

	// 构造返回结构（避免暴露 user_id 等敏感字段）
	type NoteDTO struct {
		ID            uint      `json:"id"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		FavoriteCount int       `json:"favorite_count"`
		IsFavorite    bool      `json:"is_favorite"`
		CreatedAt     time.Time `json:"created_at"`
	}

	result := make([]NoteDTO, len(notes))
	for i, n := range notes {
		result[i] = NoteDTO{
			ID:            n.ID,
			Title:         n.Title,
			Content:       n.Content,
			FavoriteCount: n.FavoriteCount,
			IsFavorite:    favSet[n.ID],
			CreatedAt:     n.CreatedAt,
		}
	}

	utils.Success(c, gin.H{"notes": result, "page": page})
}

/*非常非常高级的游标分页版，不过我还没搞懂，先封存在这
func (h *NoteHandler) ListPublicNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	sortBy := c.DefaultQuery("sort", "time") // time / popular
	cursor := c.Query("cursor")              // 游标格式：时间戳_ID
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	query := h.db.Where("is_private = ?", false)

	// 解析游标
	var cursorTime time.Time
	var cursorID uint
	if cursor != "" {
		// 游标格式：2006-01-02T15:04:05.999Z_123
		parts := strings.Split(cursor, "_")
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				cursorTime = t
			}
			if id, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
				cursorID = uint(id)
			}
		}
	}

	// 根据排序方式和游标构建查询
	switch sortBy {
	case "popular":
		// 按热度排序（收藏数）
		query = query.Order("favorite_count DESC, created_at DESC, id DESC")
		if !cursorTime.IsZero() && cursorID > 0 {
			// 我们需要知道游标记录的favorite_count
			var cursorNote models.Note
			if err := h.db.Select("favorite_count").First(&cursorNote, cursorID).Error; err == nil {
				// 复杂查询：需要获取游标笔记的收藏数
				query = query.Where(
					"(favorite_count < ?) OR "+
						"(favorite_count = ? AND created_at < ?) OR "+
						"(favorite_count = ? AND created_at = ? AND id < ?)",
					cursorNote.FavoriteCount,
					cursorNote.FavoriteCount, cursorTime,
					cursorNote.FavoriteCount, cursorTime, cursorID,
				)
			}
		}
	default: // time
		// 按时间排序
		query = query.Order("created_at DESC, id DESC")
		if !cursorTime.IsZero() && cursorID > 0 {
			// 使用复合游标：时间戳 + ID
			query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
				cursorTime, cursorTime, cursorID)
		}
	}

	var notes []models.Note
	// 多查一条，用于判断是否有更多数据
	query.Limit(limit + 1).Find(&notes)

	// 判断是否有更多数据
	hasMore := len(notes) > limit
	if hasMore {
		notes = notes[:limit] // 去掉多查的一条
	}

	// 如果没有数据，直接返回
	if len(notes) == 0 {
		utils.Success(c, gin.H{
			"notes":    []interface{}{},
			"has_more": false,
			"cursor":   "",
		})
		return
	}

	// 注入"当前用户是否已收藏"
	noteIDs := make([]uint, len(notes))
	for i, n := range notes {
		noteIDs[i] = n.ID
	}

	var favorites []models.Favorite
	h.db.Where("user_id = ? AND note_id IN ?", userID, noteIDs).Find(&favorites)
	favSet := make(map[uint]bool)
	for _, f := range favorites {
		favSet[f.NoteID] = true
	}

	// 构造返回结构
	type NoteDTO struct {
		ID            uint      `json:"id"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		FavoriteCount int       `json:"favorite_count"`
		IsFavorite    bool      `json:"is_favorite"`
		CreatedAt     time.Time `json:"created_at"`
	}

	result := make([]NoteDTO, len(notes))
	for i, n := range notes {
		result[i] = NoteDTO{
			ID:            n.ID,
			Title:         n.Title,
			Content:       n.Content,
			FavoriteCount: n.FavoriteCount,
			IsFavorite:    favSet[n.ID],
			CreatedAt:     n.CreatedAt,
		}
	}

	// 生成下一页游标（最后一条记录的信息）
	nextCursor := ""
	if hasMore && len(notes) > 0 {
		lastNote := notes[len(notes)-1]
		// 格式：时间戳_ID
		nextCursor = fmt.Sprintf("%s_%d",
			lastNote.CreatedAt.Format(time.RFC3339Nano),
			lastNote.ID)
	}

	utils.Success(c, gin.H{
		"notes":    result,
		"has_more": hasMore,
		"cursor":   nextCursor,
	})
}
*/
