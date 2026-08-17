package executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"distributed-task-queue/worker/internal/task"
)

type Executor interface {
	Execute(ctx context.Context, t task.Task) (map[string]any, error)
}

type Dispatcher struct {
	logger *slog.Logger
}

func NewDispatcher(logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Dispatcher{logger: logger}
}

func (d *Dispatcher) Execute(ctx context.Context, t task.Task) (map[string]any, error) {
	start := time.Now()
	d.logger.Debug("executing task", "task_id", t.ID, "type", t.Type)
	result, err := d.dispatch(ctx, t)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		d.logger.Debug("task execution failed", "task_id", t.ID, "type", t.Type, "duration_ms", durationMs, "error", err)
		return nil, err
	}
	d.logger.Debug("task execution succeeded", "task_id", t.ID, "type", t.Type, "duration_ms", durationMs, "result", result)
	return result, nil
}

func (d *Dispatcher) dispatch(ctx context.Context, t task.Task) (map[string]any, error) {
	switch t.Type {
	case "echo":
		return d.echo(ctx, t)
	case "sleep":
		return d.sleep(ctx, t)
	case "fibonacci":
		return d.fibonacci(ctx, t)
	default:
		return nil, fmt.Errorf("unknown task type: %s", t.Type)
	}
}