package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func redisURL() string {
	return os.Getenv("REDIS_URL")
}

func TestRedisQueuePop(t *testing.T) {
	url := redisURL()
	if url == "" {
		t.Skip("REDIS_URL is not set")
	}
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		t.Skip("redis is not available")
	}
	defer client.Close()

	key := "dtq:tasks:test"
	if err := client.Del(context.Background(), key).Err(); err != nil {
		t.Fatalf("del key: %v", err)
	}

	q, err := NewRedis(url, key)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	defer q.Close()

	if err := client.RPush(context.Background(), key, "task-1", "task-2").Err(); err != nil {
		t.Fatalf("rpush: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2 * time.Second)
	defer cancel()

	got := map[string]bool{}
	for len(got) < 2 {
		id, err := q.Pop(ctx)
		if err != nil {
			t.Fatalf("Pop: %v", err)
		}
		got[id] = true
	}
	if !got["task-1"] || !got["task-2"] {
		t.Errorf("missing ids from queue: %v", got)
	}
}

func TestRedisQueueCancel(t *testing.T) {
	url := redisURL()
	if url == "" {
		t.Skip("REDIS_URL is not set")
	}
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		t.Skip("redis is not available")
	}
	client.Close()

	key := "dtq:tasks:test"
	q, err := NewRedis(url, key)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
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
