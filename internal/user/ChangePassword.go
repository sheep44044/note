package user

import (
	"errors"
	"net/http"
	"note/internal/models"
	"note/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (h *UserHandler) ModifyPassword(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req models.PasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var user models.User
	if err := h.svc.DB.Select("password").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, http.StatusUnauthorized, "user not found")
		} else {
			zap.L().Error("db query failed", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		utils.Error(c, http.StatusUnauthorized, "old password is incorrect")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "failed to hash new password")
		return
	}

	if err := h.svc.DB.Model(&user).Update("password", string(newHash)).Error; err != nil {
		zap.L().Error("update password failed", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "failed to update password")
		return
	}

	utils.Success(c, gin.H{"message": "password changed successfully"})
}
