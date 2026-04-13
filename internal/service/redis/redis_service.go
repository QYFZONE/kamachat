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

func DelKeysWithPattern(pattern string) error {
	var cursor uint64 = 0
	var batchSize int64 = 100 // 每批扫描数量，控制单次处理量

	for {
		// 使用 SCAN 替代 KEYS，非阻塞、游标迭代
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return fmt.Errorf("scan keys failed: %w", err)
		}

		// 批量删除本批次
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				// 部分删除失败也继续，记录日志
				log.Printf("delete keys partial failed: %v, keys: %v", err, keys)
			} else {
				log.Printf("deleted %d keys with pattern %s", len(keys), pattern)
			}
		}

		cursor = nextCursor
		// 游标归零表示遍历完成
		if cursor == 0 {
			break
		}
	}

	return nil
}
