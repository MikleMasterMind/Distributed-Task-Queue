package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	TasksDir     string
	Concurrency  int
	PollInterval time.Duration
}

func Load() Config {
	tasksDir := flag.String("tasks-dir", envOrDefault("TASKS_DIR", "data/tasks"), "directory with task JSON files")
	concurrency := flag.Int("concurrency", envIntOrDefault("CONCURRENCY", 4), "number of concurrently running tasks")
	pollInterval := flag.Duration("poll-interval", time.Duration(envIntOrDefault("POLL_INTERVAL_MS", 1000))*time.Millisecond, "interval between directory scans")
	flag.Parse()
	return Config{
		TasksDir:     *tasksDir,
		Concurrency:  *concurrency,
		PollInterval: *pollInterval,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}