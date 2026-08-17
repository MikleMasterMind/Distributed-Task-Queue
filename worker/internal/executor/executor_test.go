package executor

import (
	"context"
	"testing"
	"time"

	"distributed-task-queue/worker/internal/task"
)

func TestEcho(t *testing.T) {
	d := NewDispatcher()
	res, err := d.Execute(context.Background(), task.Task{
		Type:    "echo",
		Payload: map[string]any{"message": "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["message"] != "hello" {
		t.Errorf("expected message=hello, got %v", res["message"])
	}
}

func TestEchoInvalidPayload(t *testing.T) {
	d := NewDispatcher()
	_, err := d.Execute(context.Background(), task.Task{
		Type:    "echo",
		Payload: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for missing message")
	}
}

func TestSleep(t *testing.T) {
	d := NewDispatcher()
	start := time.Now()
	res, err := d.Execute(context.Background(), task.Task{
		Type:    "sleep",
		Payload: map[string]any{"seconds": 0.0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["slept"] != 0 {
		t.Errorf("expected slept=0, got %v", res["slept"])
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("sleep(0) took too long: %v", elapsed)
	}
}

func TestSleepCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := NewDispatcher()
	_, err := d.Execute(ctx, task.Task{
		Type:    "sleep",
		Payload: map[string]any{"seconds": 5.0},
	})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestSleepInvalidPayload(t *testing.T) {
	d := NewDispatcher()
	for _, payload := range []map[string]any{
		{},
		{"seconds": -1.0},
		{"seconds": 1.5},
		{"seconds": "5"},
	} {
		if _, err := d.Execute(context.Background(), task.Task{
			Type:    "sleep",
			Payload: payload,
		}); err == nil {
			t.Errorf("expected error for payload %v", payload)
		}
	}
}

func TestFibonacci(t *testing.T) {
	d := NewDispatcher()
	res, err := d.Execute(context.Background(), task.Task{
		Type:    "fibonacci",
		Payload: map[string]any{"n": 10.0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["value"] != 55 {
		t.Errorf("expected fib(10)=55, got %v", res["value"])
	}
}

func TestFibonacciEdgeCases(t *testing.T) {
	d := NewDispatcher()
	for n, want := range map[int]int{0: 0, 1: 1, 2: 1, 20: 6765} {
		res, err := d.Execute(context.Background(), task.Task{
			Type:    "fibonacci",
			Payload: map[string]any{"n": n},
		})
		if err != nil {
			t.Fatalf("unexpected error for n=%d: %v", n, err)
		}
		if res["value"] != want {
			t.Errorf("fib(%d)=%d, want %d", n, res["value"], want)
		}
	}
}

func TestFibonacciInvalidPayload(t *testing.T) {
	d := NewDispatcher()
	for _, payload := range []map[string]any{
		{},
		{"n": -1.0},
		{"n": 1.5},
		{"n": "10"},
	} {
		if _, err := d.Execute(context.Background(), task.Task{
			Type:    "fibonacci",
			Payload: payload,
		}); err == nil {
			t.Errorf("expected error for payload %v", payload)
		}
	}
}

func TestUnknownType(t *testing.T) {
	d := NewDispatcher()
	_, err := d.Execute(context.Background(), task.Task{
		Type:    "unknown",
		Payload: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}