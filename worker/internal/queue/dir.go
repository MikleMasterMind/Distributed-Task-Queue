package queue

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type DirQueue struct {
	listPending  func(context.Context) ([]string, error)
	pollInterval time.Duration
	logger       *slog.Logger
	mu           sync.Mutex
	buffer       []string
}

func NewDir(
	listPending func(context.Context) ([]string, error),
	pollInterval time.Duration,
	logger *slog.Logger,
) *DirQueue {
	if logger == nil {
		logger = slog.Default()
	}
	return &DirQueue{
		listPending:  listPending,
		pollInterval: pollInterval,
		logger:       logger,
	}
}

func (q *DirQueue) Pop(ctx context.Context) (string, error) {
	for {
		q.mu.Lock()
		if len(q.buffer) > 0 {
			id := q.buffer[0]
			q.buffer = q.buffer[1:]
			q.mu.Unlock()
			return id, nil
		}
		ids, err := q.listPending(ctx)
		if err != nil {
			q.logger.Error("failed to scan for pending tasks", "error", err)
		} else {
			q.buffer = append(q.buffer, ids...)
		}
		empty := err != nil || len(ids) == 0
		q.mu.Unlock()
		if empty {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(q.pollInterval):
			}
		}
	}
}

func (q *DirQueue) Close() error {
	return nil
}