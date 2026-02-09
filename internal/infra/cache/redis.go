package cache

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"note/config"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func New(cfg *config.Config) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{client: rdb}, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *RedisCache) SetWithRandomTTL(ctx context.Context, key string, value interface{}, baseTTL time.Duration) error {
	// 在基础TTL上增加±10%的随机浮动
	jitter := time.Duration(rand.Int63n(int64(baseTTL/5)) - int64(baseTTL/10))
	actualTTL := baseTTL + jitter

	// 确保TTL不会变成负数
	if actualTTL < 0 {
		actualTTL = baseTTL
	}

	return c.client.Set(ctx, key, value, actualTTL).Err()
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisCache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

func (c *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

func (c *RedisCache) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	return c.client.ZAdd(ctx, key, members...).Result()
}

func (c *RedisCache) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) (int64, error) {
	return c.client.ZRemRangeByRank(ctx, key, start, stop).Result()
}

func (c *RedisCache) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return c.client.ZRem(ctx, key, members...).Result()
}

func (c *RedisCache) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.ZRevRange(ctx, key, start, stop).Result()
}

func (c *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return c.client.Expire(ctx, key, expiration).Result()
}

func (c *RedisCache) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

func (c *RedisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

func (c *RedisCache) ClearCacheByPattern(ctx context.Context, cache *RedisCache, pattern string) error {
	var cursor uint64
	var keys []string
	var err error

	// 使用 SCAN 而不是 KEYS (KEYS 会阻塞生产环境的 Redis)
	for {
		// 每次扫 100 个，防止阻塞
		keys, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			// 批量删除
			// 使用 Pipeline 提高删除效率
			pipe := cache.Pipeline()
			pipe.Del(ctx, keys...)
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}

func (c *RedisCache) AllowRequest(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Lua 脚本：计数 + 首次过期设置
	// 逻辑：如果 key 不存在，INCR 并 EXPIRE；如果存在，只 INCR
	const script = `
        local current = redis.call("INCR", KEYS[1])
        if tonumber(current) == 1 then
            redis.call("EXPIRE", KEYS[1], ARGV[1])
        end
        return current
    `

	count, err := c.client.Eval(ctx, script, []string{key}, int(window.Seconds())).Int()
	if err != nil {
		return true, err
	}

	return count <= limit, nil
}
