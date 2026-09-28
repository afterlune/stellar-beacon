package shared

import (
	"context"
	"errors"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CODE_EXPIRE_TIME    = 1000000000 * 60 * 15
	USER_CODE_KEY       = "code:"
	BLOG_VIEWS_COUNT    = "blog_views_count"
	ARTICLE_VIEWS_COUNT = "article_views_count"
	WEBSITE_CONFIG      = "website_config"
	USER_AREA           = "user_area"
	VISITOR_AREA        = "visitor_area"
	ABOUT               = "about"
	UNIQUE_VISITOR      = "unique_visitor"
	LOGIN_USER          = "login_user"
	ARTICLE_ACCESS      = "article_access:"
)

var rdb *redis.Client

// setRedisClientForTest swaps the package client and returns a restore
// function. It is intentionally small so unit tests can use an in-memory
// Redis server without depending on application configuration.
func setRedisClientForTest(client *redis.Client) func() {
	previous := rdb
	rdb = client
	return func() { rdb = previous }
}

func init() {
	redisConf := new(config.Redis).Redis()
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisConf.Addr,
		Password: redisConf.Password,
		DB:       redisConf.DB,
	})
}

func validContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func SIsMemberCtx(ctx context.Context, key string, value interface{}) (bool, error) {
	return rdb.SIsMember(validContext(ctx), key, value).Result()
}

func HIncrByCtx(ctx context.Context, key, hashKey string, delta int64) (int64, error) {
	return rdb.HIncrBy(validContext(ctx), key, hashKey, delta).Result()
}

func IncrByCtx(ctx context.Context, key string, delta int64) (int64, error) {
	return rdb.IncrBy(validContext(ctx), key, delta).Result()
}

func SAddCtx(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return rdb.SAdd(validContext(ctx), key, values...).Result()
}

func GetCtx(ctx context.Context, key string) (string, error) {
	result, err := rdb.Get(validContext(ctx), key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return result, err
}

func SetCtx(ctx context.Context, key string, value interface{}) error {
	_, err := rdb.Set(validContext(ctx), key, value, -1).Result()
	return err
}

func ZIncrCtx(ctx context.Context, key string, score float64, value string) (float64, error) {
	return rdb.ZIncrBy(validContext(ctx), key, score, value).Result()
}

func ZScoreCtx(ctx context.Context, key, value string) (float64, error) {
	result, err := rdb.ZScore(validContext(ctx), key, value).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return result, err
}

func SetWithTimeCtx(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	_, err := rdb.Set(validContext(ctx), key, value, ttl).Result()
	return err
}

// HSetCtx overwrites an existing field and updates the hash TTL atomically.
func HSetCtx(ctx context.Context, key, hashKey string, value interface{}, ttl time.Duration) error {
	ctx = validContext(ctx)
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, hashKey, value)
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

func ExpireCtx(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return rdb.Expire(validContext(ctx), key, ttl).Result()
}

func HGetCtx(ctx context.Context, key, hashKey string) (string, error) {
	result, err := rdb.HGet(validContext(ctx), key, hashKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return result, err
}

func HDelCtx(ctx context.Context, key, hashKey string) error {
	_, err := rdb.HDel(validContext(ctx), key, hashKey).Result()
	return err
}

func ZReverseRangeWithScoreCtx(ctx context.Context, key string, start, end int64) (map[interface{}]float64, error) {
	result, err := rdb.ZRevRangeWithScores(validContext(ctx), key, start, end).Result()
	if err != nil {
		return nil, err
	}
	hm := make(map[interface{}]float64, len(result))
	for _, value := range result {
		hm[value.Member] = value.Score
	}
	return hm, nil
}

func HGetAllCtx(ctx context.Context, key string) (map[string]string, error) {
	return rdb.HGetAll(validContext(ctx), key).Result()
}

func ZAllScoreCtx(ctx context.Context, key string) (map[interface{}]float64, error) {
	result, err := rdb.ZRangeWithScores(validContext(ctx), key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	hm := make(map[interface{}]float64, len(result))
	for _, value := range result {
		hm[value.Member] = value.Score
	}
	return hm, nil
}

func DelCtx(ctx context.Context, key string) error {
	_, err := rdb.Del(validContext(ctx), key).Result()
	return err
}

func IncrExpireCtx(ctx context.Context, key string, ttl time.Duration) (int64, error) {
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
	result, err := script.Run(validContext(ctx), rdb, []string{key}, seconds).Int64()
	return result, err
}

// Set is used by the background user-area refresh task.
func Set(key string, value interface{}) {
	if err := SetCtx(context.Background(), key, value); err != nil {
		logRedisError("Set", err)
	}
}

func logRedisError(operation string, err error) {
	slog.Error("redis operation failed", "operation", operation, "error", err)
}
