package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"distributed-task-queue/worker/internal/task"
)

func writeTask(t *testing.T, dir string, tk *task.Task) {
	t.Helper()
	if err := (&FileStore{dir: dir}).writeLocked(tk.ID, tk); err != nil {
		t.Fatalf("write task %s: %v", tk.ID, err)
	}
}

func makeTask(id string, status task.Status) *task.Task {
	return &task.Task{
		ID:        id,
		Type:      "echo",
		Payload:   map[string]any{"message": "hello"},
		Status:    status,
		CreatedAt: time.Now().UTC(),
	}
}

func TestListPending(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	writeTask(t, dir, makeTask("pending", task.StatusPending))
	writeTask(t, dir, makeTask("running", task.StatusRunning))
	writeTask(t, dir, makeTask("success", task.StatusSuccess))
	writeTask(t, dir, makeTask("failed", task.StatusFailed))

	ids, err := s.ListPending(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "pending" {
		t.Errorf("expected [pending], got %v", ids)
	}
}

func TestListPendingMissingDir(t *testing.T) {
	s := NewFileStore(t.TempDir() + "/nope")
	ids, err := s.ListPending(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty list, got %v", ids)
	}
}

func TestClaim(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	writeTask(t, dir, makeTask("t1", task.StatusPending))

	started := time.Now().UTC()
	claimed, err := s.Claim(context.Background(), "t1", started)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claimed.Status != task.StatusRunning {
		t.Errorf("expected RUNNING, got %s", claimed.Status)
	}
	if claimed.StartedAt == nil || !claimed.StartedAt.Equal(started) {
		t.Errorf("started_at not set correctly: %v", claimed.StartedAt)
	}

	onDisk, err := (&FileStore{dir: dir}).readLocked("t1")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if onDisk.Status != task.StatusRunning {
		t.Errorf("expected RUNNING on disk, got %s", onDisk.Status)
	}
}

func TestClaimNotPending(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	writeTask(t, dir, makeTask("t1", task.StatusSuccess))
	_, err := s.Claim(context.Background(), "t1", time.Now().UTC())
	if !errors.Is(err, ErrNotPending) {
		t.Errorf("expected ErrNotPending, got %v", err)
	}
}

func TestClaimMissing(t *testing.T) {
	s := NewFileStore(t.TempDir())
	_, err := s.Claim(context.Background(), "missing", time.Now().UTC())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestComplete(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	writeTask(t, dir, makeTask("t1", task.StatusPending))
	if _, err := s.Claim(context.Background(), "t1", time.Now().UTC()); err != nil {
		t.Fatalf("claim: %v", err)
	}

	finished := time.Now().UTC()
	result := map[string]any{"message": "done"}
	if err := s.Complete(context.Background(), "t1", result, finished); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	onDisk, err := (&FileStore{dir: dir}).readLocked("t1")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if onDisk.Status != task.StatusSuccess {
		t.Errorf("expected SUCCESS, got %s", onDisk.Status)
	}
	if onDisk.Result["message"] != "done" {
		t.Errorf("expected result message=done, got %v", onDisk.Result)
	}
	if onDisk.Error != nil {
		t.Errorf("expected nil error, got %v", *onDisk.Error)
	}
	if onDisk.FinishedAt == nil || !onDisk.FinishedAt.Equal(finished) {
		t.Errorf("finished_at not set correctly: %v", onDisk.FinishedAt)
	}
}

func TestFail(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	writeTask(t, dir, makeTask("t1", task.StatusPending))
	if _, err := s.Claim(context.Background(), "t1", time.Now().UTC()); err != nil {
		t.Fatalf("claim: %v", err)
	}

	finished := time.Now().UTC()
	if err := s.Fail(context.Background(), "t1", "boom", finished); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	onDisk, err := (&FileStore{dir: dir}).readLocked("t1")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if onDisk.Status != task.StatusFailed {
		t.Errorf("expected FAILED, got %s", onDisk.Status)
	}
	if onDisk.Error == nil || *onDisk.Error != "boom" {
		t.Errorf("expected error=boom, got %v", onDisk.Error)
	}
	if onDisk.FinishedAt == nil || !onDisk.FinishedAt.Equal(finished) {
		t.Errorf("finished_at not set correctly: %v", onDisk.FinishedAt)
	}
}

func TestConcurrentClaimSingleWinner(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	writeTask(t, dir, makeTask("t1", task.StatusPending))

	const workers = 8
	var wg sync.WaitGroup
	winners := make(chan string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Claim(context.Background(), "t1", time.Now().UTC()); err == nil {
				winners <- "won"
			}
		}()
	}
	wg.Wait()
	close(winners)

	if count := len(winners); count != 1 {
		t.Errorf("expected exactly 1 winner, got %d", count)
	}
}