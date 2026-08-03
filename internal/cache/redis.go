package cache

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
	once   sync.Once
)

func InitializeRedis() {
	once.Do(func() {
		redisURL := os.Getenv("REDIS_URL")
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "redis:6379"
		}

		var opts *redis.Options
		var err error

		if redisURL != "" {
			opts, err = redis.ParseURL(redisURL)
			if err != nil {
				log.Printf("Redis unavailable, falling back to database: invalid REDIS_URL: %v", err)
				return
			}
		} else {
			opts = &redis.Options{Addr: redisAddr}
		}

		client = redis.NewClient(opts)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			log.Printf("Redis unavailable, falling back to database: %v", err)
			_ = client.Close()
			client = nil
			return
		}
		log.Printf("Redis connected")
	})
}

func GetClient() *redis.Client {
	return client
}
