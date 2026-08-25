package queue

import (
	"fmt"
	"log/slog"
)

type QueueConfig struct {
	Kind  string
	Redis RedisConfig
}

type RedisConfig struct {
	URL string
	Key string
}

func New(cfg QueueConfig, logger *slog.Logger) (Queue, error) {
	switch cfg.Kind {
	case "redis":
		return NewRedis(cfg.Redis.URL, cfg.Redis.Key)
	default:
		return nil, fmt.Errorf("unknown queue type %q", cfg.Kind)
	}
}
