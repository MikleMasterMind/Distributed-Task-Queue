package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"distributed-task-queue/worker/internal/config"
	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/queue"
	"distributed-task-queue/worker/internal/store"
	"distributed-task-queue/worker/internal/worker"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := store.NewFileStore(cfg.TasksDir, logger)
	e := executor.NewDispatcher(logger)

	q, err := queue.NewQueue(cfg, s, logger)
	if err != nil {
		logger.Error("failed to create queue", "error", err)
		os.Exit(1)
	}

	w := worker.New(s, e, q, cfg.Concurrency, logger)

	logger.Info("worker started", "queue", cfg.QueueType, "tasks_dir", cfg.TasksDir, "concurrency", cfg.Concurrency, "poll_interval", cfg.PollInterval.String())
	w.Run(ctx)
	logger.Info("worker stopped")
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
