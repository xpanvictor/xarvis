package database

import (
	"github.com/go-redis/redis"
	"github.com/xpanvictor/xarvis/internal/config"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Pass, // Add via env if needed
		DB:       0,
	})
	return client, nil
}
