package queue

import (
	"context"
	"log/slog"

	"distributed-task-queue/worker/internal/config"
	"distributed-task-queue/worker/internal/store"
)

type Queue interface {
	Pop(ctx context.Context) (string, error)
	Close() error
}

func NewQueue(cfg config.Config, s *store.FileStore, logger *slog.Logger) (Queue, error) {
	return New(
		QueueConfig{
			Kind: cfg.QueueType,
			Redis: RedisConfig{
				URL: cfg.RedisURL,
				Key: cfg.QueueKey,
			},
			Dir: DirConfig{
				ListPending:  s.ListPending,
				PollInterval: cfg.PollInterval,
			},
		},
		logger,
	)
}
