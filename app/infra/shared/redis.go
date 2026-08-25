package shared

import (
	"benetnasch/app/infra/config"
	"context"
	"errors"
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

func SetNXCtx(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return rdb.SetNX(validContext(ctx), key, value, ttl).Result()
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

// The following wrappers are kept for non-HTTP legacy callers. New code must
// use the Context variants above so request cancellation reaches Redis.
func SIsMember(key string, value interface{}) bool {
	result, err := SIsMemberCtx(context.Background(), key, value)
	if err != nil {
		logRedisError("SIsMember", err)
	}
	return result
}

func HIncrBy(key, hashKey string, delta int64) int64 {
	result, err := HIncrByCtx(context.Background(), key, hashKey, delta)
	if err != nil {
		logRedisError("HIncrBy", err)
	}
	return result
}

func IncrBy(key string, delta int64) int64 {
	result, err := IncrByCtx(context.Background(), key, delta)
	if err != nil {
		logRedisError("IncrBy", err)
	}
	return result
}

func SAdd(key string, values ...interface{}) int64 {
	result, err := SAddCtx(context.Background(), key, values...)
	if err != nil {
		logRedisError("SAdd", err)
	}
	return result
}

func Get(key string) interface{} {
	result, err := GetCtx(context.Background(), key)
	if err != nil {
		logRedisError("Get", err)
	}
	return result
}

func Set(key string, value interface{}) {
	if err := SetCtx(context.Background(), key, value); err != nil {
		logRedisError("Set", err)
	}
}

func ZIncr(key string, score float64, value string) float64 {
	result, err := ZIncrCtx(context.Background(), key, score, value)
	if err != nil {
		logRedisError("ZIncr", err)
	}
	return result
}

func ZScore(key, value string) float64 {
	result, err := ZScoreCtx(context.Background(), key, value)
	if err != nil {
		logRedisError("ZScore", err)
	}
	return result
}

func SetWithTime(key string, value interface{}, ttl time.Duration) {
	if err := SetWithTimeCtx(context.Background(), key, value, ttl); err != nil {
		logRedisError("SetWithTime", err)
	}
}

func HSet(key, hashKey string, value interface{}, ttl time.Duration) bool {
	if err := HSetCtx(context.Background(), key, hashKey, value, ttl); err != nil {
		logRedisError("HSet", err)
		return false
	}
	return true
}

func Expire(key string, ttl time.Duration) bool {
	result, err := ExpireCtx(context.Background(), key, ttl)
	if err != nil {
		logRedisError("Expire", err)
	}
	return result
}

func HGet(key, hashKey string) string {
	result, err := HGetCtx(context.Background(), key, hashKey)
	if err != nil {
		logRedisError("HGet", err)
	}
	return result
}

func HDel(key, hashKey string) {
	if err := HDelCtx(context.Background(), key, hashKey); err != nil {
		logRedisError("HDel", err)
	}
}

func ZReverseRangeWithScore(key string, start, end int64) map[interface{}]float64 {
	result, err := ZReverseRangeWithScoreCtx(context.Background(), key, start, end)
	if err != nil {
		logRedisError("ZReverseRangeWithScore", err)
	}
	return result
}

func HGetAll(key string) map[string]string {
	result, err := HGetAllCtx(context.Background(), key)
	if err != nil {
		logRedisError("HGetAll", err)
	}
	return result
}

func ZAllScore(key string) map[interface{}]float64 {
	result, err := ZAllScoreCtx(context.Background(), key)
	if err != nil {
		logRedisError("ZAllScore", err)
	}
	return result
}

func Del(key string) {
	if err := DelCtx(context.Background(), key); err != nil {
		logRedisError("Del", err)
	}
}

func IncrExpire(key string, ttl time.Duration) int64 {
	result, err := IncrExpireCtx(context.Background(), key, ttl)
	if err != nil {
		logRedisError("IncrExpire", err)
	}
	return result
}

func logRedisError(operation string, err error) {
	slog.Error("redis operation failed", "operation", operation, "error", err)
}
