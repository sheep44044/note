package user

import (
	"errors"
	"log/slog"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *UserHandler) PersonalPage(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 查用户基本信息
	var user models.User
	if err := h.db.Select("id, username, avatar, bio, created_at, updated_at").
		Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "user not found")
		} else {
			slog.Error("DB query user error")
			utils.Error(c, http.StatusInternalServerError, "internal error")
		}
		return
	}

	// 查该用户的所有笔记（只查公开的 or 自己的）
	var notes []models.Note

	err = h.db.Select("id, title, is_private, created_at, updated_at").
		Where("user_id = ? AND is_private = ?", user.ID, false).
		Or("user_id = ? AND is_private = ? AND user_id = ?", user.ID, true, user.ID).
		Find(&notes).Error

	if err != nil {
		slog.Error("DB query notes error")
		utils.Error(c, http.StatusInternalServerError, "failed to load notes")
		return
	}

	// 转换为响应结构
	noteBriefs := make([]models.NoteBrief, len(notes))
	for i, n := range notes {
		noteBriefs[i] = models.NoteBrief{
			ID:        n.ID,
			Title:     n.Title,
			IsPrivate: n.IsPrivate,
			CreatedAt: n.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: n.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	page := models.PersonalPage{
		ID:        user.ID,
		Username:  user.Username,
		Avatar:    user.Avatar,
		Bio:       user.Bio,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		Documents: noteBriefs,
	}

	utils.Success(c, page)
}

func (h *UserHandler) UpdateMyProfile(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req validators.UpdateProfileRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
	}

	// 3. 构造要更新的字段（只更新非 nil 的）
	updates := make(map[string]interface{})
	if req.Username != nil {
		*req.Username = strings.TrimSpace(*req.Username)
		if *req.Username == "" {
			utils.Error(c, http.StatusBadRequest, "用户名不能为空")
			return
		}
		updates["username"] = *req.Username
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Bio != nil {
		updates["bio"] = strings.TrimSpace(*req.Bio)
	}

	if len(updates) == 0 {
		utils.Error(c, http.StatusBadRequest, "至少需要提供一个要修改的字段")
		return
	}

	// 4. 执行更新（只更新当前用户的记录）
	result := h.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(updates)

	if result.Error != nil {
		slog.Error("Update profile DB error", "error", result.Error)
		utils.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}

	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 5. 返回更新后的完整信息（可选）
	var updatedUser models.User
	h.db.Select("id, username, avatar, bio, created_at, updated_at").
		First(&updatedUser, userID)

	response := map[string]interface{}{
		"id":         updatedUser.ID,
		"username":   updatedUser.Username,
		"avatar":     updatedUser.Avatar,
		"bio":        updatedUser.Bio,
		"created_at": updatedUser.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at": updatedUser.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	utils.Success(c, response)
}
