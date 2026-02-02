package user

import (
	"errors"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (h *UserHandler) PersonalPage(c *gin.Context) {
	viewerID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}

	param := c.Param("id")
	var targetID uint

	if param == "me" {
		targetID = viewerID
	} else {
		idUint64, err := strconv.ParseUint(param, 10, 64)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "无效的用户ID")
			return
		}
		targetID = uint(idUint64)
	}

	isOwner := viewerID == targetID

	var targetUser models.User
	err = h.db.Select("id, username, avatar, bio, follow_count, fan_count, created_at").
		Where("id = ?", targetID).
		First(&targetUser).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "用户不存在")
		} else {
			zap.L().Error("db query user failed", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "获取用户信息失败")
		}
		return
	}

	var notes []models.Note
	query := h.db.Preload("Tags").
		Select("id, title, summary, is_private, is_pinned, favorite_count, created_at, updated_at").
		Where("user_id = ?", targetID).
		Order("is_pinned DESC, updated_at DESC")

	if !isOwner {
		query = query.Where("is_private = ?", false)
	}

	if err := query.Find(&notes).Error; err != nil {
		zap.L().Error("db query notes failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "获取笔记列表失败")
		return
	}

	isFollowing := false
	if !isOwner {
		var count int64
		h.db.Model(&models.UserFollow{}).
			Where("follower_id = ? AND followed_id = ?", viewerID, targetID).
			Count(&count)
		isFollowing = count > 0
	}

	noteBriefs := make([]models.NoteBrief, len(notes))
	for i, n := range notes {
		var tagNames []string
		for _, t := range n.Tags {
			tagNames = append(tagNames, t.Name)
		}
		noteBriefs[i] = models.NoteBrief{
			ID:            n.ID,
			Title:         n.Title,
			Summary:       n.Summary,
			FavoriteCount: n.FavoriteCount,
			IsPrivate:     n.IsPrivate,
			IsPinned:      n.IsPinned,
			Tags:          tagNames,
			UpdatedAt:     n.UpdatedAt.Format("2006-01-02"),
		}
	}

	page := models.PersonalPage{
		ID:          targetUser.ID,
		Username:    targetUser.Username,
		Avatar:      targetUser.Avatar,
		Bio:         targetUser.Bio,
		FollowCount: targetUser.FollowCount,
		FanCount:    targetUser.FanCount,
		IsFollowing: isFollowing,
		CreatedAt:   targetUser.CreatedAt.Format("2006-01-02"),
		Documents:   noteBriefs,
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
		return
	}

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

	result := h.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(updates)

	if result.Error != nil {
		zap.L().Error("Update profile DB error", zap.Error(result.Error))
		utils.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}

	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, "用户不存在")
		return
	}

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
