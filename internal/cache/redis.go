package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(addr, password string, db int, ttl time.Duration) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisCache{client: client, ttl: ttl}
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Get(key string) ([]byte, bool) {
	ctx := context.Background()
	value, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return value, true
}

func (c *RedisCache) Set(key string, value []byte) {
	ctx := context.Background()
	_ = c.client.Set(ctx, key, value, c.ttl).Err()
}

