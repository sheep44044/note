package note

import (
	"log/slog"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *NoteHandler) SearchNotes(c *gin.Context) {
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
	// 1. 获取查询参数
	query := c.Query("q")
	if query == "" {
		utils.Error(c, http.StatusBadRequest, "缺少搜索关键词 'q'")
		return
	}

	// 2. 安全限制：关键词长度 ≤ 50 字符（防滥用）
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

	// 4. 构建 LIKE 查询（GORM 自动转义，防注入）
	var notes []models.Note
	offset := (page - 1) * pageSize

	// 同时搜标题和内容（忽略大小写）
	err := h.db.Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%").
		Where("user_id = ?", userID). // ← 别忘了权限！
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&notes).Error

	if err != nil {
		slog.Error("Search notes failed", "error", err)
		utils.Error(c, http.StatusInternalServerError, "搜索失败")
		return
	}

	utils.Success(c, gin.H{
		"notes": notes,
		"page":  page,
		"total": len(notes),
	})
}
