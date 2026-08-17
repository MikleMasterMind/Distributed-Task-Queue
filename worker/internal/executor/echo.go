package executor

import (
	"context"
	"errors"

	"distributed-task-queue/worker/internal/task"
)

func (d *Dispatcher) echo(ctx context.Context, t task.Task) (map[string]any, error) {
	msg, ok := t.Payload["message"].(string)
	if !ok {
		return nil, errors.New("payload.message must be a string")
	}
	return map[string]any{"message": msg}, nil
}