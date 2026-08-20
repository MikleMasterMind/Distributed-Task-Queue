package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/queue"
	"distributed-task-queue/worker/internal/store"
)

type Worker struct {
	store        store.Store
	exec         executor.Executor
	logger       *slog.Logger
	workersCount int
	queue        queue.Queue
}

func New(s store.Store, e executor.Executor, q queue.Queue, workersCount int, logger *slog.Logger) *Worker {
	if workersCount < 1 {
		workersCount = 1
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		store:        s,
		exec:         e,
		logger:       logger,
		workersCount: workersCount,
		queue:        q,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer w.queue.Close()

	w.logger.Debug("starting workers", "count", w.workersCount)
	var wg sync.WaitGroup
	for i := 0; i < w.workersCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.consume(ctx)
		}()
	}

	<-ctx.Done()
	w.logger.Info("shutdown: waiting for running tasks")
	wg.Wait()
	w.logger.Debug("all workers finished")
}

func (w *Worker) consume(ctx context.Context) {
	for {
		id, err := w.queue.Pop(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			w.logger.Error("failed to pop task from queue", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		w.process(id)
	}
}

func (w *Worker) process(id string) {
	t, err := w.store.Claim(context.Background(), id, time.Now().UTC())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrNotPending) {
			w.logger.Warn("task claim skipped", "task_id", id, "reason", err)
			return
		}
		w.logger.Error("failed to claim task", "task_id", id, "error", err)
		return
	}

	w.logger.Info("task claimed", "task_id", id, "type", t.Type)
	start := time.Now()
	result, err := w.exec.Execute(context.Background(), t)
	durationMs := time.Since(start).Milliseconds()
	finishedAt := time.Now().UTC()

	if err != nil {
		w.logger.Error("task failed", "task_id", id, "type", t.Type, "duration_ms", durationMs, "error", err)
		if serr := w.store.Fail(context.Background(), id, err.Error(), finishedAt); serr != nil {
			w.logger.Error("failed to persist task failure", "task_id", id, "error", serr)
		}
		return
	}

	w.logger.Info("task succeeded", "task_id", id, "type", t.Type, "duration_ms", durationMs, "result", result)
	if serr := w.store.Complete(context.Background(), id, result, finishedAt); serr != nil {
		w.logger.Error("failed to persist task completion", "task_id", id, "error", serr)
	}
}