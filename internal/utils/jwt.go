package utils

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"note/config"
	"note/internal/cache"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func GenerateToken(cfg *config.Config, userID uint, username string) (string, error) {
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

func IsTokenBlacklisted(tokenString string) (bool, error) {
	// 先简单解析token获取jti，不验证签名（因为要先检查黑名单）
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false, nil
	}

	// 只解析claims部分
	claims := jwt.MapClaims{}
	_, _, _ = jwt.NewParser().ParseUnverified(tokenString, claims) //三个返回值是完整的token、[]string分开的token和err

	// 3. 安全提取 jti（兼容 string 和 float64）
	var jtiStr string
	if jti, ok := claims["jti"].(string); ok {
		jtiStr = jti
	} else if jti, ok := claims["jti"].(float64); ok {
		jtiStr = fmt.Sprintf("%d", int64(jti))
	} else {
		// 没有 jti 或类型不对，无法加入黑名单
		return false, nil
	}

	// 4. 查询 Redis 黑名单
	key := "blacklist:" + jtiStr
	_, err := cache.Get(key)
	// 不在黑名单中
	if errors.Is(err, redis.Nil) {
		return false, nil
	}

	if err != nil {
		//  Redis 出错了！返回错误，由调用方决定是否降级
		return false, fmt.Errorf("redis error checking blacklist: %w", err)
	}

	return true, nil
}

func AddTokenToBlacklist(tokenString string, expiration time.Duration) error {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if jti, ok := claims["jti"].(float64); ok {
		key := "blacklist:" + fmt.Sprintf("%d", int64(jti))
		return cache.Set(key, "1", expiration)
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
