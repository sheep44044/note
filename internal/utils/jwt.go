package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"note/config"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func GenerateToken(cfg *config.Config, userID string, username string) (string, error) {
	// 生成唯一ID用于黑名单
	jti := time.Now().UnixNano() + rand.Int63()

	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"jti":      jti,
		"exp":      time.Now().Add(cfg.JWTExpirationTime).Unix(),
		"iat":      time.Now().Unix(),
		"iss":      cfg.JWTIssuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecretKey))
}

// 检查token是否在黑名单中
func IsTokenBlacklisted(redisClient *redis.Client, tokenString string) (bool, error) {
	// 先简单解析token获取jti，不验证签名（因为要先检查黑名单）
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false, nil
	}

	// 只解析claims部分
	claims := jwt.MapClaims{}
	_, _, _ = jwt.NewParser().ParseUnverified(tokenString, claims)

	// 3. 安全提取 jti（兼容 string 和 float64）
	var jtiStr string
	if jti, ok := claims["jti"].(string); ok {
		jtiStr = jti
	} else if jti, ok := claims["jti"].(float64); ok {
		jtiStr = strconv.FormatInt(int64(jti), 10)
	} else {
		// 没有 jti 或类型不对，无法加入黑名单
		return false, nil
	}

	// 4. 查询 Redis 黑名单
	key := "blacklist:" + jtiStr
	_, err := redisClient.Get(context.Background(), key).Result()

	if err == redis.Nil {
		// 不在黑名单中
		return false, nil
	}
	if err != nil {
		// 🔥 Redis 出错了！返回错误，由调用方决定是否降级
		return false, fmt.Errorf("redis error checking blacklist: %w", err)
	}
	// 存在即被拉黑
	return true, nil
}

// 将token加入黑名单
func AddTokenToBlacklist(redisClient *redis.Client, tokenString string, expiration time.Duration) error {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if jti, ok := claims["jti"].(float64); ok {
		key := "blacklist:" + strconv.FormatInt(int64(jti), 10)
		return redisClient.Set(context.Background(), key, "1", expiration).Err()
	}
	return nil
}

func ValidateToken(cfg *config.Config, tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecretKey), nil
	})
}

func ExtractClaims(token *jwt.Token) (jwt.MapClaims, error) {
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func GetTokenHash(token string) string {
	if token == "" {
		return "empty"
	}
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash[:8]) // 取前8字节（16字符）足够区分，又不冗长
}
