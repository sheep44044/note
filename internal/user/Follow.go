package user

import (
	"errors"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (h *UserHandler) FollowUser(c *gin.Context) {
	targetIDstr := c.Param("id")
	targetIDUint64, err := strconv.ParseUint(targetIDstr, 10, 64)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid target ID")
		return
	}
	targetID := uint(targetIDUint64)

	me, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	if me == targetID {
		utils.Error(c, http.StatusBadRequest, "You can't follow yourself")
		return
	}

	var exists int64
	h.svc.DB.Model(&models.User{}).Where("id = ?", targetID).Count(&exists)
	if exists == 0 {
		utils.Error(c, http.StatusNotFound, "用户不存在")
		return
	}

	err = h.svc.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&models.UserFollow{}).
			Where("follower_id = ? AND followed_id = ?", me, targetID).
			Count(&count)
		if count > 0 {
			return errors.New("already_followed")
		}

		followRel := models.UserFollow{
			FollowedID: targetID,
			FollowerID: me,
		}

		if err := tx.Create(&followRel).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.User{}).Where("id = ?", me).
			Update("follow_count", gorm.Expr("follow_count + 1")).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.User{}).Where("id = ?", targetID).
			Update("fan_count", gorm.Expr("fan_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if err.Error() == "already_followed" {
			utils.Error(c, http.StatusBadRequest, "已关注该用户")
		} else {
			zap.L().Error("follow user failed", zap.Error(err), zap.Uint("me", me), zap.Uint("target", targetID))
			utils.Error(c, http.StatusInternalServerError, "关注失败")
		}
		return
	}
	utils.Success(c, gin.H{"message": "Followed successfully"})
}

func (h *UserHandler) UnfollowUser(c *gin.Context) {
	targetIDstr := c.Param("id")
	targetIDUint64, err := strconv.ParseUint(targetIDstr, 10, 64)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid target ID")
		return
	}
	targetID := uint(targetIDUint64)

	me, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	err = h.svc.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("follower_id = ? AND followed_id = ?", me, targetID).Delete(&models.UserFollow{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return nil
		}

		if err := tx.Model(&models.User{}).Where("id = ?", me).
			Update("follow_count", gorm.Expr("follow_count - 1")).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.User{}).Where("id = ?", targetID).
			Update("fan_count", gorm.Expr("fan_count - 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		zap.L().Error("unfollow user failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "取消关注失败")
		return
	}
	utils.Success(c, gin.H{"message": "Unfollowed successfully"})
}

func (h *UserHandler) GetFollowingList(c *gin.Context) {
	targetID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	offset := (page - 1) * size

	var users []models.UserBrief
	err := h.svc.DB.Table("users").
		Select("users.id, users.username, users.avatar, users.bio").
		Joins("JOIN user_follows ON users.id = user_follows.followed_id").
		Where("user_follows.follower_id = ?", targetID).
		Limit(size).Offset(offset).
		Scan(&users).Error

	if err != nil {
		zap.L().Error("get following list failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "获取列表失败")
		return
	}

	utils.Success(c, users)
}

func (h *UserHandler) GetFollowersList(c *gin.Context) {
	targetID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	offset := (page - 1) * size

	var users []models.UserBrief

	err := h.svc.DB.Table("users").
		Select("users.id, users.username, users.avatar, users.bio").
		Joins("JOIN user_follows ON users.id = user_follows.follower_id").
		Where("user_follows.followed_id = ?", targetID).
		Limit(size).Offset(offset).
		Scan(&users).Error

	if err != nil {
		zap.L().Error("get followers list failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "获取列表失败")
		return
	}

	utils.Success(c, users)
}
