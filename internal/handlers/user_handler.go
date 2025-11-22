package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"note/config"
	"note/internal/models"
	"note/internal/redis1"
	"note/internal/utils"
	"note/internal/validators"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db  *gorm.DB
	rdb *redis.Client
	cfg *config.Config
}

func NewUserHandler(db *gorm.DB, cfg *config.Config, rdb *redis.Client) *UserHandler {
	return &UserHandler{db: db, cfg: cfg, rdb: rdb}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req validators.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var exists models.User
	if h.db.Where("username = ?", req.Username).First(&exists).RowsAffected > 0 {
		utils.Error(c, http.StatusConflict, "username already exists")
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{
		Username: req.Username,
		Password: string(hashed),
	}
	h.db.Create(&user)

	utils.Success(c, gin.H{"message": "user registered"})
}

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

	userIDStr := strconv.FormatUint(uint64(user.ID), 10)
	token, err := utils.GenerateToken(h.cfg, userIDStr, user.Username)
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
		cacheKey := "user:session:" + userIDStr
		expiration := h.cfg.JWTExpirationTime // 使用与JWT相同的过期时间

		if err := redis1.Set(cacheKey, string(userDataJSON), expiration); err != nil {
			slog.Warn("failed to cache user session", "error", err, "user_id", user.ID)
			// 注意：缓存失败不应阻止登录成功，只记录警告
		} else {
			slog.Debug("user session cached successfully", "user_id", user.ID, "cache_key", cacheKey)
		}
	}
	// ===== Redis缓存结束 =====

	utils.Success(c, gin.H{"token": token, "user": gin.H{
		"id":       user.ID,
		"username": user.Username,
	}})
}
