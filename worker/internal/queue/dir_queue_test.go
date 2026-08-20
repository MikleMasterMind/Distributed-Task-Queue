package queue

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"distributed-task-queue/worker/internal/store"
	"distributed-task-queue/worker/internal/task"
)

func TestDirQueuePopReturnsPending(t *testing.T) {
	dir := t.TempDir()
	s := store.NewFileStore(dir, testLogger)
	writeTask(t, dir, "a")
	writeTask(t, dir, "b")

	q := NewDir(s.ListPending, 5*time.Millisecond, testLogger)
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got := map[string]bool{}
	for len(got) < 2 {
		id, err := q.Pop(ctx)
		if err != nil {
			t.Fatalf("Pop: %v", err)
		}
		got[id] = true
	}
	if !got["a"] || !got["b"] {
		t.Errorf("missing ids from queue: %v", got)
	}
}

func TestDirQueueCancel(t *testing.T) {
	dir := t.TempDir()
	s := store.NewFileStore(dir, testLogger)
	q := NewDir(s.ListPending, 10*time.Millisecond, testLogger)
	defer q.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	if _, err := q.Pop(ctx); err == nil {
		t.Fatal("expected error on cancel")
	}
	if time.Since(start) > time.Second {
		t.Error("Pop did not return promptly on cancel")
	}
}

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func writeTask(t *testing.T, dir string, id string) {
	t.Helper()
	data, err := json.Marshal(task.Task{
		ID:        id,
		Type:      "echo",
		Payload:   map[string]any{"message": "hello"},
		Status:    task.StatusPending,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}