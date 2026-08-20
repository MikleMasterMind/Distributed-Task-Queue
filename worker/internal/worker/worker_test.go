package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"distributed-task-queue/worker/internal/executor"
	"distributed-task-queue/worker/internal/queue"
	"distributed-task-queue/worker/internal/store"
	"distributed-task-queue/worker/internal/task"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func writeTask(t *testing.T, dir, id, typ string, payload map[string]any) {
	t.Helper()
	data, err := json.Marshal(task.Task{
		ID:        id,
		Type:      typ,
		Payload:   payload,
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

func readTask(t *testing.T, dir, id string) task.Task {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var tk task.Task
	if err := json.Unmarshal(data, &tk); err != nil {
		t.Fatalf("unmarshal task %s: %v", id, err)
	}
	return tk
}

func startWorker(t *testing.T, dir string, concurrency int) (*Worker, context.CancelFunc) {
	t.Helper()
	s := store.NewFileStore(dir, testLogger)
	e := executor.NewDispatcher(testLogger)
	q := queue.NewDir(s.ListPending, 5*time.Millisecond, testLogger)
	w := New(s, e, q, concurrency, testLogger)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return w, cancel
}

func waitFinal(t *testing.T, dir string, ids []string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		all := true
		for _, id := range ids {
			status := readTask(t, dir, id).Status
			if status != task.StatusSuccess && status != task.StatusFailed {
				all = false
				break
			}
		}
		if all {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for tasks %v to finish", ids)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestWorkerProcessesTasks(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, "t1", "echo", map[string]any{"message": "a"})
	writeTask(t, dir, "t2", "echo", map[string]any{"message": "b"})
	writeTask(t, dir, "t3", "fibonacci", map[string]any{"n": 10})

	startWorker(t, dir, 2)
	waitFinal(t, dir, []string{"t1", "t2", "t3"})

	t1 := readTask(t, dir, "t1")
	if t1.Status != task.StatusSuccess || t1.Result["message"] != "a" {
		t.Errorf("t1 wrong outcome: %+v", t1)
	}
	if t1.StartedAt == nil || t1.FinishedAt == nil {
		t.Errorf("t1 timestamps missing: %+v", t1)
	}
	t3 := readTask(t, dir, "t3")
	if t3.Result["value"] != float64(55) {
		t.Errorf("expected fib(10)=55, got %v", t3.Result["value"])
	}
}

func TestWorkerFailureDoesNotStopOthers(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, "ok1", "echo", map[string]any{"message": "a"})
	writeTask(t, dir, "bad", "echo", map[string]any{"message": 123})
	writeTask(t, dir, "ok2", "echo", map[string]any{"message": "b"})

	startWorker(t, dir, 2)
	waitFinal(t, dir, []string{"ok1", "bad", "ok2"})

	if got := readTask(t, dir, "ok1").Status; got != task.StatusSuccess {
		t.Errorf("ok1 expected SUCCESS, got %s", got)
	}
	bad := readTask(t, dir, "bad")
	if bad.Status != task.StatusFailed {
		t.Errorf("bad expected FAILED, got %s", bad.Status)
	}
	if bad.Error == nil || *bad.Error == "" {
		t.Errorf("bad expected error message, got %v", bad.Error)
	}
	if got := readTask(t, dir, "ok2").Status; got != task.StatusSuccess {
		t.Errorf("ok2 expected SUCCESS, got %s", got)
	}
}

func TestWorkerDoesNotRunTaskTwice(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, "t1", "sleep", map[string]any{"seconds": 0})

	startWorker(t, dir, 4)
	waitFinal(t, dir, []string{"t1"})

	t1 := readTask(t, dir, "t1")
	if t1.Status != task.StatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", t1.Status)
	}
	time.Sleep(30 * time.Millisecond)
	if got := readTask(t, dir, "t1").Status; got != task.StatusSuccess {
		t.Errorf("task was re-run: got %s", got)
	}
}

func TestGracefulShutdown(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, "slow", "sleep", map[string]any{"seconds": 2})

	s := store.NewFileStore(dir, testLogger)
	e := executor.NewDispatcher(testLogger)
	q := queue.NewDir(s.ListPending, 5*time.Millisecond, testLogger)
	w := New(s, e, q, 1, testLogger)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	waitRunning := func() {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for {
			if readTask(t, dir, "slow").Status == task.StatusRunning {
				return
			}
			if time.Now().After(deadline) {
				t.Fatal("task never became RUNNING")
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	waitRunning()

	start := time.Now()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not shut down")
	}

	if elapsed := time.Since(start); elapsed < 1900*time.Millisecond {
		t.Errorf("worker did not wait for running task, stopped after %v", elapsed)
	}
	if got := readTask(t, dir, "slow").Status; got != task.StatusSuccess {
		t.Errorf("slow task expected SUCCESS, got %s", got)
	}

	writeTask(t, dir, "after", "echo", map[string]any{"message": "x"})
	time.Sleep(50 * time.Millisecond)
	if got := readTask(t, dir, "after").Status; got != task.StatusPending {
		t.Errorf("worker accepted a new task after shutdown, got %s", got)
	}
}

func TestWorkerLogsTaskResult(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, "t1", "echo", map[string]any{"message": "hello"})

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	s := store.NewFileStore(dir, logger)
	e := executor.NewDispatcher(logger)
	q := queue.NewDir(s.ListPending, 5*time.Millisecond, logger)
	w := New(s, e, q, 1, logger)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()
	waitFinal(t, dir, []string{"t1"})
	cancel()
	<-done

	logs := buf.String()
	if !strings.Contains(logs, "task succeeded") {
		t.Errorf("expected log 'task succeeded', got:\n%s", logs)
	}
	if !strings.Contains(logs, "task_id=t1") {
		t.Errorf("expected log with task_id=t1, got:\n%s", logs)
	}
}