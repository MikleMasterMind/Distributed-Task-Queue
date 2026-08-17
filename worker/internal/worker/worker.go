package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/store"
)

type Worker struct {
	store        store.Store
	exec         executor.Executor
	logger       *slog.Logger
	concurrency  int
	pollInterval time.Duration
	tasks        chan string
}

func New(s store.Store, e executor.Executor, concurrency int, pollInterval time.Duration, logger *slog.Logger) *Worker {
	if concurrency < 1 {
		concurrency = 1
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		store:        s,
		exec:         e,
		logger:       logger,
		concurrency:  concurrency,
		pollInterval: pollInterval,
		tasks:        make(chan string),
	}
}

func (w *Worker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	w.logger.Debug("starting poller")
	var poller sync.WaitGroup
	poller.Add(1)
	go func() {
		defer poller.Done()
		w.poll(ctx)
	}()

	w.logger.Debug("starting workers", "count", w.concurrency)
	var workers sync.WaitGroup
	for i := 0; i < w.concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			w.consume()
		}()
	}

	<-ctx.Done()
	w.logger.Info("shutdown: waiting for running tasks")
	poller.Wait()
	w.logger.Debug("poller stopped")
	close(w.tasks)
	workers.Wait()
	w.logger.Debug("all workers finished")
}

func (w *Worker) poll(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	w.scan(ctx)
	for {
		select {
		case <-ticker.C:
			w.scan(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) scan(ctx context.Context) {
	ids, err := w.store.ListPending(ctx)
	if err != nil {
		w.logger.Error("failed to scan for pending tasks", "error", err)
		return
	}
	if len(ids) > 0 {
		w.logger.Debug("found pending tasks", "count", len(ids))
	}
	for _, id := range ids {
		select {
		case w.tasks <- id:
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) consume() {
	for id := range w.tasks {
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