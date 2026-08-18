package store

import (
	"context"
	"errors"
	"time"

	"distributed-task-queue/worker/internal/task"
)

var (
	ErrNotFound    = errors.New("task not found")
	ErrNotPending  = errors.New("task is not pending")
)

type Store interface {
	ListPending(ctx context.Context) ([]string, error)
	Claim(ctx context.Context, id string, startedAt time.Time) (task.Task, error)
	Complete(ctx context.Context, id string, result map[string]any, finishedAt time.Time) error
	Fail(ctx context.Context, id string, errMsg string, finishedAt time.Time) error
}