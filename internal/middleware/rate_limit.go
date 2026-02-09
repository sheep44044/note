package middleware

import (
	"fmt"
	"log"
	"net/http"
	"note/internal/infra/cache"
	"note/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimitMiddleware(rdb *cache.RedisCache, action string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserID(c)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		key := fmt.Sprintf("rate:limit:%v:%s", userID, action)

		allowed, err := rdb.AllowRequest(c, key, limit, window)
		if err != nil {
			log.Printf("[RateLimit Error] Redis failed for key %s: %v", key, err)
			c.Next()
			return
		}

		if !allowed {
			utils.Error(c, http.StatusTooManyRequests, "操作太频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
