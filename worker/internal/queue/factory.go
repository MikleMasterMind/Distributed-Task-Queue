package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type QueueConfig struct {
	Kind string

	Redis RedisConfig
	Dir   DirConfig
}

type RedisConfig struct {
	URL string
	Key string
}

type DirConfig struct {
	ListPending  func(context.Context) ([]string, error)
	PollInterval time.Duration
}

func New(cfg QueueConfig, logger *slog.Logger) (Queue, error) {
	switch cfg.Kind {
	case "redis":
		return NewRedis(cfg.Redis.URL, cfg.Redis.Key)
	case "dir":
		return NewDir(cfg.Dir.ListPending, cfg.Dir.PollInterval, logger), nil
	default:
		return nil, fmt.Errorf("unknown queue type %q", cfg.Kind)
	}
}
