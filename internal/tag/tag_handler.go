package tag

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/svc"
	"note/internal/utils"
	"note/internal/validators"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NoteTag struct {
	svc *svc.ServiceContext
}

func NewNoteTag(svc *svc.ServiceContext) *NoteTag {
	return &NoteTag{svc: svc}
}

func (h *NoteTag) GetTags(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	cacheKey := fmt.Sprintf("tags:user:%d", userID)
	cachedTags, err := h.svc.Cache.Get(c, cacheKey)
	if err == nil {
		var tags []models.Tag
		if err := json.Unmarshal([]byte(cachedTags), &tags); err == nil {
			zap.L().Debug("Tags retrieved from cache", zap.String("key", cacheKey))
			utils.Success(c, tags)
			return
		}
	}

	var tags []models.Tag
	if err := h.svc.DB.Where("user_id = ?", userID).Find(&tags).Error; err != nil {
		zap.L().Error("Failed to fetch tags DB", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "获取标签失败")
		return
	}

	tagsJSON, _ := json.Marshal(tags)
	_ = h.svc.Cache.SetWithRandomTTL(c, cacheKey, string(tagsJSON), 10*time.Minute)

	utils.Success(c, tags)
}

func (h *NoteTag) GetTag(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	id := c.Param("id")
	cacheKey := "tag:" + id

	cachedTag, err := h.svc.Cache.Get(c, cacheKey)
	if err == nil {
		var tag models.Tag
		if err := json.Unmarshal([]byte(cachedTag), &tag); err == nil {
			if tag.UserID == userID {
				zap.L().Debug("Tag retrieved from cache", zap.String("key", cacheKey))
				utils.Success(c, tag)
				return
			}
		}
	}

	var tag models.Tag
	if err := h.svc.DB.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "tag not found")
		} else {
			zap.L().Error("db error", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	tagJSON, _ := json.Marshal(tag)
	_ = h.svc.Cache.SetWithRandomTTL(c, cacheKey, string(tagJSON), 10*time.Minute)

	utils.Success(c, tag)
}

func (h *NoteTag) CreateTag(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	var req validators.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "invalid tag")
		return
	}

	var count int64
	h.svc.DB.Model(&models.Tag{}).Where("user_id = ? AND name = ?", userID, req.Name).Count(&count)
	if count > 0 {
		utils.Error(c, http.StatusBadRequest, "Tag name already exists")
		return
	}

	tag := models.Tag{
		Name:   req.Name,
		Color:  req.Color,
		UserID: userID,
	}
	if err := h.svc.DB.Create(&tag).Error; err != nil {
		zap.L().Error("create tag db error", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}

	_ = h.svc.Cache.Del(c, fmt.Sprintf("tags:user:%d", userID))

	utils.Success(c, tag)
}

func (h *NoteTag) UpdateTag(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	id := c.Param("id")
	var req validators.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var tag models.Tag
	if err := h.svc.DB.Where("id = ? AND user_id = ?", id, userID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusNotFound, "tag not found")
		} else {
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	if err := h.svc.DB.Model(&tag).Updates(models.Tag{
		Name:  req.Name,
		Color: req.Color,
	}).Error; err != nil {
		utils.Error(c, http.StatusBadRequest, "更新失败，可能标签名已存在")
		return
	}

	_ = h.svc.Cache.Del(c, "tag:"+id)
	_ = h.svc.Cache.Del(c, fmt.Sprintf("tags:user:%d", userID))

	zap.L().Info("Cache cleared for updated tag", zap.String("tag_id", id))

	utils.Success(c, tag)
}

func (h *NoteTag) DeleteTag(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		utils.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	result := h.svc.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Tag{})

	if result.Error != nil {
		zap.L().Error("delete tag db error", zap.Error(result.Error))
		utils.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}

	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, "tag not found")
		return
	}

	_ = h.svc.Cache.Del(c, "tag:"+c.Param("id"))
	_ = h.svc.Cache.Del(c, fmt.Sprintf("tags:user:%d", userID))

	zap.L().Info("Tag and related caches cleared", zap.Int("tag_id", id))
	utils.Success(c, gin.H{"message": "deleted"})
}
