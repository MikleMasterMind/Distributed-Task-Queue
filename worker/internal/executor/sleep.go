package executor

import (
	"context"
	"errors"
	"time"

	"distributed-task-queue/worker/internal/task"
)

func (d *Dispatcher) sleep(ctx context.Context, t task.Task) (map[string]any, error) {
	seconds, err := asNonNegativeInt(t.Payload["seconds"])
	if err != nil {
		return nil, errors.New("payload.seconds must be a non-negative integer")
	}
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		return map[string]any{"slept": seconds}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}