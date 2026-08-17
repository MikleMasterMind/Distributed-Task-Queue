package executor

import (
	"context"
	"fmt"

	"distributed-task-queue/worker/internal/task"
)

type Executor interface {
	Execute(ctx context.Context, t task.Task) (map[string]any, error)
}

type Dispatcher struct{}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) Execute(ctx context.Context, t task.Task) (map[string]any, error) {
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