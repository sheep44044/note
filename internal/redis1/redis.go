package redis1

import (
	"context"
	"fmt"
	"math/rand"
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

func SetWithRandomTTL(key string, value interface{}, baseTTL time.Duration) error {
	// 在基础TTL上增加±10%的随机浮动
	jitter := time.Duration(rand.Int63n(int64(baseTTL/5)) - int64(baseTTL/10))
	actualTTL := baseTTL + jitter

	// 确保TTL不会变成负数
	if actualTTL < 0 {
		actualTTL = baseTTL
	}

	return Rdb.Set(ctx, key, value, actualTTL).Err()
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

func ZAdd(key string, members ...redis.Z) (int64, error) {
	return Rdb.ZAdd(ctx, key, members...).Result()
}

func ZRemRangeByRank(key string, start, stop int64) (int64, error) {
	return Rdb.ZRemRangeByRank(ctx, key, start, stop).Result()
}

func ZRem(key string, members ...interface{}) (int64, error) {
	return Rdb.ZRem(ctx, key, members...).Result()
}

func Expire(key string, expiration time.Duration) (bool, error) {
	return Rdb.Expire(ctx, key, expiration).Result()
}
