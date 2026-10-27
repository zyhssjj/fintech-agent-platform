package database

import (
	"log/slog"

	"github.com/redis/go-redis/v9"

	"wfagent/internal/base/config"
)

// NewRedis initializes the shared Redis client used by cache, token blacklist,
// rate limiting, and async task infrastructure.
func NewRedis() *redis.Client {
	cfg := config.C.Redis
	if !cfg.Enabled {
		slog.Warn("redis disabled; cache-backed base features will run in degraded mode")
		return nil
	}

	return redis.NewClient(&redis.Options{
		Addr:         cfg.Address(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
}
