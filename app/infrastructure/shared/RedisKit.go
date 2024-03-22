package shared

import (
	"benetnasch/app/infrastructure/config"
	"benetnasch/app/infrastructure/exception"
	"context"
	"github.com/redis/go-redis/v9"
	"log"
	"sync"
	"time"
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

var rdbOnce sync.Once
var rdb *redis.Client

func init() {
	rdbOnce.Do(func() {
		redisConf := new(config.Redis).Redis()
		rdb = redis.NewClient(&redis.Options{
			Addr:     redisConf.Addr,
			Password: redisConf.Password,
			DB:       redisConf.DB,
		})
	})
}

func SIsMember(key string, value interface{}) bool {
	ctx := context.Background()
	result, _ := rdb.SIsMember(ctx, key, value).Result()
	return result
}

func HIncrBy(key, hashkey string, delta int64) int64 {
	ctx := context.Background()
	result, err := rdb.HIncrBy(ctx, key, hashkey, delta).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return result
}
func IncrBy(key string, delta int64) int64 {
	ctx := context.Background()
	result, err := rdb.IncrBy(ctx, key, delta).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return result
}
func SAdd(key string, values ...interface{}) int64 {
	ctx := context.Background()
	result, err := rdb.SAdd(ctx, key, values).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return result
}

func Get(key string) interface{} {
	ctx := context.Background()
	result, _ := rdb.Get(ctx, key).Result()
	return result
}

func Set(key string, value interface{}) {
	ctx := context.Background()
	_, err := rdb.Set(ctx, key, value, -1).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
}

func ZIncr(key string, score float64, value string) float64 {
	ctx := context.Background()
	result, err := rdb.ZIncrBy(ctx, key, score, value).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return result
}

func ZScore(key string, value string) float64 {
	ctx := context.Background()
	result, err := rdb.ZScore(ctx, key, value).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return result
}

func SetWithTime(key string, value interface{}, time time.Duration) {
	ctx := context.Background()
	rdb.Set(ctx, key, value, time)
}

func HSet(key, hashKey string, value interface{}, time time.Duration) bool {
	ctx := context.Background()
	_, err := rdb.HSetNX(ctx, key, hashKey, value).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return Expire(key, time)
}

func Expire(key string, time time.Duration) bool {
	ctx := context.Background()
	result, err := rdb.Expire(ctx, key, time).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
	}
	return result
}

func HGet(key, hashKey string) string {
	ctx := context.Background()
	result, err := rdb.HGet(ctx, key, hashKey).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
		return ""
	}
	return result
}

func HDel(key, hashKey string) {
	ctx := context.Background()
	rdb.HDel(ctx, key, hashKey)
}

func ZReverseRangeWithScore(key string, start, end int64) map[interface{}]float64 {
	ctx := context.Background()
	result, err := rdb.ZRevRangeWithScores(ctx, key, start, end).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
		return nil
	}
	hm := make(map[interface{}]float64)
	for _, v := range result {
		hm[v.Member] = v.Score
	}
	return hm
}

func HGetAll(key string) map[string]string {
	ctx := context.Background()
	result, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
		return nil
	}
	return result
}

func ZAllScore(key string) map[interface{}]float64 {
	ctx := context.Background()
	result, err := rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
		return nil
	}
	hm := make(map[interface{}]float64)
	for _, v := range result {
		hm[v.Member] = v.Score
	}
	return hm
}

func Del(key string) {
	ctx := context.Background()
	rdb.Del(ctx, key)
}

func IncrExpire(key string, time time.Duration) int64 {
	ctx := context.Background()
	count, err := rdb.IncrBy(ctx, key, 1).Result()
	if err != nil {
		exception.Logger.Println(err)
		exception.PrintStack()
		log.Println(err)
		return -1
	}
	if count == 1 {
		Expire(key, time)
	}
	return count
}
