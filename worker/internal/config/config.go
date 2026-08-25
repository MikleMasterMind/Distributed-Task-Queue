package config

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	DBAutoCreate bool
	StoreType    string
	Concurrency  int
	PollInterval time.Duration
	LogLevel     string
	QueueType    string
	RedisURL     string
	QueueKey     string
}

func Load() Config {
	_ = godotenv.Load()
	databaseURL := flag.String("database-url", envOrDefault("DATABASE_URL", ""), "PostgreSQL connection URL")
	dbAutoCreate := flag.Bool("db-auto-create", envBoolOrDefault("DB_AUTO_CREATE", true), "auto-create database tables")
	storeType := flag.String("store", envOrDefault("STORE_TYPE", "postgres"), "store backend: postgres, file")
	concurrency := flag.Int("concurrency", envIntOrDefault("CONCURRENCY", 4), "number of concurrently running tasks")
	pollInterval := flag.Duration("poll-interval", time.Duration(envIntOrDefault("POLL_INTERVAL_MS", 1000))*time.Millisecond, "interval between polls")
	logLevel := flag.String("log-level", envOrDefault("LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	queueType := flag.String("queue", envOrDefault("QUEUE_TYPE", "redis"), "queue backend: redis, dir")
	redisURL := flag.String("redis-url", envOrDefault("REDIS_URL", "redis://localhost:6379/0"), "redis connection URL")
	queueKey := flag.String("queue-key", envOrDefault("QUEUE_KEY", "dtq:tasks"), "redis list key for the task queue")
	flag.Parse()
	return Config{
		DatabaseURL:  *databaseURL,
		DBAutoCreate: *dbAutoCreate,
		StoreType:    *storeType,
		Concurrency:  *concurrency,
		PollInterval: *pollInterval,
		LogLevel:     *logLevel,
		QueueType:    *queueType,
		RedisURL:     *redisURL,
		QueueKey:     *queueKey,
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

func envBoolOrDefault(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
