package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is the infrastructure implementation of the application cache
// port. It owns a single go-redis client and preserves context cancellation
// for every operation.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(conf *config.Redis) *RedisCache {
	return NewRedisCacheWithClient(redis.NewClient(&redis.Options{
		Addr:     conf.Addr,
		Password: conf.Password,
		DB:       conf.DB,
	}))
}

func NewRedisCacheWithClient(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func cacheContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(cacheContext(ctx), key).Result()
	if errors.Is(err, redis.Nil) {
		return "", port.ErrCacheMiss
	}
	return value, err
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	_, err := r.client.Set(cacheContext(ctx), key, value, ttl).Result()
	return err
}

func (r *RedisCache) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return r.client.SetNX(cacheContext(ctx), key, value, ttl).Result()
}

func (r *RedisCache) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	seconds := int64(ttl / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	script := redis.NewScript(`
local count = redis.call('INCRBY', KEYS[1], 1)
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)
	return script.Run(cacheContext(ctx), r.client, []string{key}, seconds).Int64()
}

func (r *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.client.Expire(cacheContext(ctx), key, ttl).Result()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	_, err := r.client.Del(cacheContext(ctx), key).Result()
	return err
}

func (r *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	value, err := r.client.HGet(cacheContext(ctx), key, field).Result()
	if errors.Is(err, redis.Nil) {
		return "", port.ErrCacheMiss
	}
	return value, err
}

func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(cacheContext(ctx), key).Result()
}

// HSet overwrites an existing field and refreshes the hash TTL atomically.
func (r *RedisCache) HSet(ctx context.Context, key, field string, value any, ttl time.Duration) error {
	ctx = cacheContext(ctx)
	_, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, field, value)
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

func (r *RedisCache) HDel(ctx context.Context, key, field string) error {
	_, err := r.client.HDel(cacheContext(ctx), key, field).Result()
	return err
}

func (r *RedisCache) HIncrBy(ctx context.Context, key, field string, delta int64) (int64, error) {
	return r.client.HIncrBy(cacheContext(ctx), key, field, delta).Result()
}

func (r *RedisCache) SIsMember(ctx context.Context, key string, value any) (bool, error) {
	return r.client.SIsMember(cacheContext(ctx), key, value).Result()
}

func (r *RedisCache) SAdd(ctx context.Context, key string, values ...any) (int64, error) {
	return r.client.SAdd(cacheContext(ctx), key, values...).Result()
}

func (r *RedisCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(cacheContext(ctx), key, delta).Result()
}

func (r *RedisCache) ZIncrBy(ctx context.Context, key string, score float64, value string) (float64, error) {
	return r.client.ZIncrBy(cacheContext(ctx), key, score, value).Result()
}

func (r *RedisCache) ZScore(ctx context.Context, key, value string) (float64, error) {
	score, err := r.client.ZScore(cacheContext(ctx), key, value).Result()
	if errors.Is(err, redis.Nil) {
		return 0, port.ErrCacheMiss
	}
	return score, err
}

func (r *RedisCache) ZRevRangeWithScores(ctx context.Context, key string, start, end int64) (map[string]float64, error) {
	values, err := r.client.ZRevRangeWithScores(cacheContext(ctx), key, start, end).Result()
	if err != nil {
		return nil, err
	}
	return scoresByMember(values), nil
}

func (r *RedisCache) ZRangeWithScores(ctx context.Context, key string) (map[string]float64, error) {
	values, err := r.client.ZRangeWithScores(cacheContext(ctx), key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	return scoresByMember(values), nil
}

func scoresByMember(values []redis.Z) map[string]float64 {
	result := make(map[string]float64, len(values))
	for _, value := range values {
		result[fmt.Sprint(value.Member)] = value.Score
	}
	return result
}

var _ port.Cache = (*RedisCache)(nil)
