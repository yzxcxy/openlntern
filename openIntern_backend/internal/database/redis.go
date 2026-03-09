package database

import (
	"context"
	"time"

	"openIntern/internal/config"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func InitRedis(cfg config.RedisConfig) error {
	if cfg.Addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}
	redisClient = client
	return nil
}

func GetRedis() *redis.Client {
	return redisClient
}
