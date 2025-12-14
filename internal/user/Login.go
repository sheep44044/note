package user

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (h *UserHandler) Login(c *gin.Context) {
	var req validators.LoginUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var user models.User
	if h.db.Where("username = ?", req.Username).First(&user).RowsAffected == 0 {
		utils.Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := utils.GenerateToken(h.cfg, user.ID, user.Username)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	userData := map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
		"token":      token, // 也可以存储token，便于后续验证
	}

	// 将用户数据转换为JSON
	userDataJSON, err := json.Marshal(userData)
	if err != nil {
		slog.Warn("failed to marshal user data for caching", "error", err, "user_id", user.ID)
	} else {
		// 使用redis包中定义的Set函数缓存数据
		// 缓存键格式: user:session:{userID}
		cacheKey := fmt.Sprintf("user:session:%d", user.ID)
		expiration := h.cfg.JWTExpirationTime // 使用与JWT相同的过期时间

		if err := h.cache.SetWithRandomTTL(c, cacheKey, string(userDataJSON), expiration); err != nil {
			slog.Warn("failed to cache user session", "error", err, "user_id", user.ID)
		} else {
			slog.Debug("user session cached successfully", "user_id", user.ID, "cache_key", cacheKey)
		}
	}

	utils.Success(c, gin.H{"token": token, "user": gin.H{
		"id":       user.ID,
		"username": user.Username,
	}})
}
