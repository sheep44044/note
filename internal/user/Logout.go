package user

import (
	"net/http"
	"note/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *UserHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		utils.Error(c, http.StatusBadRequest, "missing token")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		utils.Error(c, http.StatusBadRequest, "invalid token format")
		return
	}
	tokenString := parts[1]

	expiration := h.svc.Config.JWTExpirationTime

	err := utils.AddTokenToBlacklist(tokenString, expiration)
	if err != nil {
		zap.L().Error("failed to add token to blacklist", zap.Error(err), zap.String("token_part", utils.GetTokenHash(tokenString)))
		utils.Error(c, http.StatusInternalServerError, "failed to logout")
		return
	}

	utils.Success(c, gin.H{"message": "logged out successfully"})
}
