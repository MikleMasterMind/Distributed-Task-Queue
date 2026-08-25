package worker

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/store"
	"distributed-task-queue/worker/internal/task"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func getTestDBURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping PostgreSQL tests")
	}
	return url
}

func createTestPostgresStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewPostgresStore(getTestDBURL(t), true, testLogger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func createPendingTaskPostgres(t *testing.T, s store.Store, id, typ string, payload map[string]any) {
	t.Helper()
	ps := s.(*store.PostgresStore)
	model := store.TaskModel{
		ID:        id,
		Type:      typ,
		Payload:   payload,
		Status:    string(task.StatusPending),
		CreatedAt: time.Now().UTC(),
	}
	result := ps.DB().WithContext(context.Background()).Create(&model)
	if result.Error != nil {
		t.Fatalf("failed to create task %s: %v", id, result.Error)
	}
}

func getTaskStatusPostgres(t *testing.T, s store.Store, id string) task.Status {
	t.Helper()
	ps := s.(*store.PostgresStore)
	var model store.TaskModel
	result := ps.DB().WithContext(context.Background()).Where("id = ?", id).First(&model)
	if result.Error != nil {
		t.Fatalf("failed to get task %s: %v", id, result.Error)
	}
	return task.Status(model.Status)
}

func getTaskResultPostgres(t *testing.T, s store.Store, id string) map[string]any {
	t.Helper()
	ps := s.(*store.PostgresStore)
	var model store.TaskModel
	result := ps.DB().WithContext(context.Background()).Where("id = ?", id).First(&model)
	if result.Error != nil {
		t.Fatalf("failed to get task %s: %v", id, result.Error)
	}
	return model.Result
}

func TestPostgresWorkerProcessesTasks(t *testing.T) {
	s := createTestPostgresStore(t)
	createPendingTaskPostgres(t, s, "t1", "echo", map[string]any{"message": "a"})
	createPendingTaskPostgres(t, s, "t2", "echo", map[string]any{"message": "b"})
	createPendingTaskPostgres(t, s, "t3", "fibonacci", map[string]any{"n": float64(10)})

	e := executor.NewDispatcher(testLogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(s, e, &noopQueue{
		ids: []string{"t1", "t2", "t3"},
	}, 2, testLogger)

	go w.Run(ctx)
	time.Sleep(2 * time.Second)

	if got := getTaskStatusPostgres(t, s, "t1"); got != task.StatusSuccess {
		t.Errorf("t1 expected SUCCESS, got %s", got)
	}
	if got := getTaskStatusPostgres(t, s, "t2"); got != task.StatusSuccess {
		t.Errorf("t2 expected SUCCESS, got %s", got)
	}
	if got := getTaskStatusPostgres(t, s, "t3"); got != task.StatusSuccess {
		t.Errorf("t3 expected SUCCESS, got %s", got)
	}
	if got := getTaskResultPostgres(t, s, "t3"); got["value"] != float64(55) {
		t.Errorf("expected fib(10)=55, got %v", got["value"])
	}
}

func TestPostgresWorkerFailureDoesNotStopOthers(t *testing.T) {
	s := createTestPostgresStore(t)
	createPendingTaskPostgres(t, s, "ok1", "echo", map[string]any{"message": "a"})
	createPendingTaskPostgres(t, s, "bad", "echo", map[string]any{"message": float64(123)})
	createPendingTaskPostgres(t, s, "ok2", "echo", map[string]any{"message": "b"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(s, executor.NewDispatcher(testLogger), &noopQueue{
		ids: []string{"ok1", "bad", "ok2"},
	}, 2, testLogger)

	go w.Run(ctx)
	time.Sleep(2 * time.Second)

	if got := getTaskStatusPostgres(t, s, "ok1"); got != task.StatusSuccess {
		t.Errorf("ok1 expected SUCCESS, got %s", got)
	}
	if got := getTaskStatusPostgres(t, s, "bad"); got != task.StatusFailed {
		t.Errorf("bad expected FAILED, got %s", got)
	}
	if got := getTaskStatusPostgres(t, s, "ok2"); got != task.StatusSuccess {
		t.Errorf("ok2 expected SUCCESS, got %s", got)
	}
}

type noopQueue struct {
	ids    []string
	idx    int
	closed bool
}

func (q *noopQueue) Pop(ctx context.Context) (string, error) {
	if q.idx >= len(q.ids) {
		<-ctx.Done()
		return "", ctx.Err()
	}
	id := q.ids[q.idx]
	q.idx++
	return id, nil
}

func (q *noopQueue) Close() error {
	q.closed = true
	return nil
}
