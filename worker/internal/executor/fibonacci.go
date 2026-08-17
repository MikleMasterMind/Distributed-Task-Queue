package executor

import (
	"context"
	"errors"

	"distributed-task-queue/worker/internal/task"
)

func (d *Dispatcher) fibonacci(ctx context.Context, t task.Task) (map[string]any, error) {
	n, err := asNonNegativeInt(t.Payload["n"])
	if err != nil {
		return nil, errors.New("payload.n must be a non-negative integer")
	}
	return map[string]any{"n": n, "value": fib(n)}, nil
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}