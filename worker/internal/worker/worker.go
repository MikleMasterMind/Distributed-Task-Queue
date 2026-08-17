package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/store"
)

type Worker struct {
	store        store.Store
	exec         executor.Executor
	concurrency  int
	pollInterval time.Duration
	tasks        chan string
}

func New(s store.Store, e executor.Executor, concurrency int, pollInterval time.Duration) *Worker {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Worker{
		store:        s,
		exec:         e,
		concurrency:  concurrency,
		pollInterval: pollInterval,
		tasks:        make(chan string),
	}
}

func (w *Worker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var poller sync.WaitGroup
	poller.Add(1)
	go func() {
		defer poller.Done()
		w.poll(ctx)
	}()

	var workers sync.WaitGroup
	for i := 0; i < w.concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			w.consume()
		}()
	}

	<-ctx.Done()
	poller.Wait()
	close(w.tasks)
	workers.Wait()
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
		return
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
			return
		}
		return
	}
	result, err := w.exec.Execute(context.Background(), t)
	finishedAt := time.Now().UTC()
	if err != nil {
		w.store.Fail(context.Background(), id, err.Error(), finishedAt)
		return
	}
	w.store.Complete(context.Background(), id, result, finishedAt)
}