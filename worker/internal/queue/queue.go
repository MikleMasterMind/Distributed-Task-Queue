package queue

import (
	"context"
	"log/slog"

	"distributed-task-queue/worker/internal/config"
)

type Queue interface {
	Pop(ctx context.Context) (string, error)
	Close() error
}

func NewQueue(cfg config.Config, logger *slog.Logger) (Queue, error) {
	return New(
		QueueConfig{
			Kind: cfg.QueueType,
			Redis: RedisConfig{
				URL: cfg.RedisURL,
				Key: cfg.QueueKey,
			},
		},
		logger,
	)
}
