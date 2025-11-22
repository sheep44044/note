package redis

import (
	"context"
	"fmt"
	"time"

	"note/config"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	ctx = context.Background()
)

func Init(cfg *config.Config) error {
	// 创建Redis客户端
	Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// 测试连接
	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// 设置键值对，带过期时间
func Set(key string, value interface{}, expiration time.Duration) error {
	return Rdb.Set(ctx, key, value, expiration).Err()
}

// 获取键值
func Get(key string) (string, error) {
	return Rdb.Get(ctx, key).Result()
}

// 删除键
func Del(key string) error {
	return Rdb.Del(ctx, key).Err()
}

// 设置哈希
func HSet(key string, values ...interface{}) error {
	return Rdb.HSet(ctx, key, values...).Err()
}

// 获取哈希
func HGet(key, field string) (string, error) {
	return Rdb.HGet(ctx, key, field).Result()
}
