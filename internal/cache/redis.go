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
		configSource := "REDIS_ADDR"

		if redisURL != "" {
			opts, err = redis.ParseURL(redisURL)
			if err != nil {
				log.Printf("Redis unavailable, falling back to database: invalid REDIS_URL: %v", err)
				return
			}
			configSource = "REDIS_URL"
		} else {
			opts = &redis.Options{Addr: redisAddr}
		}

		log.Printf("Redis config source: %s", configSource)
		log.Printf("Redis config: addr=%s db=%d", opts.Addr, opts.DB)

		client = redis.NewClient(opts)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pong, err := client.Ping(ctx).Result()
		log.Printf("Redis ping: %s err=%v", pong, err)
		if err != nil {
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
