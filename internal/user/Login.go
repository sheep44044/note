package user

import (
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (h *UserHandler) Login(c *gin.Context) {
	var req validators.LoginUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var user models.User
	if h.svc.DB.Where("username = ?", req.Username).First(&user).RowsAffected == 0 {
		utils.Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := utils.GenerateToken(h.svc.Config, user.ID, user.Username)
	if err != nil {
		zap.L().Error("failed to generate token", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	utils.Success(c, gin.H{"token": token, "user": gin.H{
		"id":       user.ID,
		"username": user.Username,
	}})
}
