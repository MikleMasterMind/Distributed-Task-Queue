package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"distributed-task-queue/worker/internal/config"
	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/store"
	"distributed-task-queue/worker/internal/worker"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := store.NewFileStore(cfg.TasksDir)
	e := executor.NewDispatcher()
	w := worker.New(s, e, cfg.Concurrency, cfg.PollInterval)

	log.Printf("worker started: tasks_dir=%s concurrency=%d poll_interval=%s", cfg.TasksDir, cfg.Concurrency, cfg.PollInterval)
	w.Run(ctx)
	log.Println("worker stopped")
}