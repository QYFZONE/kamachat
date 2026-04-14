package redis

import (
	"context"
	"errors"
	"fmt"
	"kama_chat_server/internal/config"
	"kama_chat_server/pkg/zlog"
	"log"
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

func DelKey(key string) error {
	return redisClient.Del(ctx, key).Err()
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

// DelKeysWithPrefix 删除所有指定前缀的 key
func DelKeysWithPrefix(prefix string) error {
	var cursor uint64
	const batchSize int64 = 100
	pattern := prefix + "*"

	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return fmt.Errorf("scan keys by prefix failed: %w", err)
		}

		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete keys by prefix failed: %w", err)
			}
			log.Printf("deleted %d keys with prefix %s: %v", len(keys), prefix, keys)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DeleteAllRedisKeys 删除当前 Redis 数据库中的所有 key
func DeleteAllRedisKeys() error {
	return redisClient.FlushDB(ctx).Err()
}
