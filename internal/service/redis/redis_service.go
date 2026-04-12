package redis

import (
	"context"
	"errors"
	"kama_chat_server/internal/config"
	"kama_chat_server/pkg/zlog"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client
var ctx = context.Background()

func init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.Host
	port := conf.RedisConfig.Port
	password := conf.RedisConfig.Password
	db := conf.Db
	addr := host + ":" + strconv.Itoa(port)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func GetKey(key string) (string, error) {
	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zlog.Info("redis key not found")
			return "", nil
		}
		return "", err
	}
	return val, err
}

func DelKeyIfExists(key string) error {
	exists, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 { // 键存在
		delErr := redisClient.Del(ctx, key).Err()
		if delErr != nil {
			return delErr
		}
	}
	// 无论键是否存在，都不返回错误
	return nil
}

func SetKeyEx(key string, code string, timeout time.Duration) error {
	err := redisClient.Set(ctx, key, code, timeout).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetKeyNilIsErr(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}
