package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	TasksDir     string
	Concurrency  int
	PollInterval time.Duration
	LogLevel     string
}

func Load() Config {
	dir := defaultTasksDir()
	if dir == "" {
		dir = "data/tasks"
	}
	tasksDir := flag.String("tasks-dir", envOrDefault("TASKS_DIR", dir), "directory with task JSON files")
	concurrency := flag.Int("concurrency", envIntOrDefault("CONCURRENCY", 4), "number of concurrently running tasks")
	pollInterval := flag.Duration("poll-interval", time.Duration(envIntOrDefault("POLL_INTERVAL_MS", 1000))*time.Millisecond, "interval between directory scans")
	logLevel := flag.String("log-level", envOrDefault("LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	flag.Parse()
	return Config{
		TasksDir:     *tasksDir,
		Concurrency:  *concurrency,
		PollInterval: *pollInterval,
		LogLevel:     *logLevel,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultTasksDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if filepath.Base(wd) == "worker" {
		return filepath.Join(filepath.Dir(wd), "data", "tasks")
	}
	return filepath.Join(wd, "data", "tasks")
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}