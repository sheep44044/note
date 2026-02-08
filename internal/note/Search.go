package note

import (
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *NoteHandler) SearchNotes(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	query := c.Query("q")
	if query == "" {
		utils.Error(c, http.StatusBadRequest, "缺少搜索关键词 'q'")
		return
	}

	if len(query) > 50 {
		utils.Error(c, http.StatusBadRequest, "搜索词过长")
		return
	}

	// 3. 分页（可选）
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 10
	offset := (page - 1) * pageSize

	keywordQuery := "%" + query + "%"
	dbQuery := h.svc.DB.Model(&models.Note{}).
		Where("title LIKE ? OR content LIKE ?", keywordQuery, keywordQuery).
		Where(h.svc.DB.Where("user_id = ?", userID).Or("is_private = ?", false))

	// 先查总数 (用于前端分页)
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "搜索失败")
		return
	}

	var notes []models.Note
	err = dbQuery.Preload("Tags").
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		zap.L().Error("Search notes failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "数据库错误")
		return
	}

	utils.Success(c, gin.H{
		"notes": notes,
		"page":  page,
		"total": len(notes),
	})
}

func (h *NoteHandler) SmartSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.Error(c, http.StatusBadRequest, "搜索内容不能为空")
		return
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	queryVec, err := h.svc.AI.GetEmbedding(query)
	if err != nil {
		zap.L().Error("AI Embedding failed", zap.Error(err))
		utils.Error(c, 500, "AI 服务繁忙")
		return
	}

	// 2. 去 Qdrant 搜出最相似的 Top 20 个 Note ID
	noteIDs, err := h.svc.Qdrant.Search(c, queryVec, 20, userID)
	if err != nil {
		utils.Error(c, 500, "搜索服务繁忙")
		return
	}

	if len(noteIDs) == 0 {
		utils.Success(c, []models.Note{})
		return
	}

	var notes []models.Note
	err = h.svc.DB.Where("id IN ?", noteIDs).
		Where(
			h.svc.DB.Where("user_id = ?", userID).
				Or("is_private = ?", false),
		).
		Find(&notes).Error

	if err != nil {
		utils.Error(c, 500, "数据库查询失败")
		return
	}

	noteMap := make(map[uint]models.Note)
	for _, n := range notes {
		noteMap[n.ID] = n
	}

	sortedNotes := make([]models.Note, 0, len(notes))
	for _, id := range noteIDs {
		if n, ok := noteMap[id]; ok {
			sortedNotes = append(sortedNotes, n)
		}
	}

	utils.Success(c, notes)
}
