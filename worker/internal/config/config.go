package config

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	DBAutoCreate bool
	StoreType    string
	Concurrency  int
	LogLevel     string
	QueueType    string
	RedisURL     string
	QueueKey     string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}
	databaseURL := flag.String("database-url", envOrDefault("DATABASE_URL", ""), "PostgreSQL connection URL")
	dbAutoCreate := flag.Bool("db-auto-create", envBoolOrDefault("DB_AUTO_CREATE", true), "auto-create database tables")
	storeType := flag.String("store", envOrDefault("STORE_TYPE", "postgres"), "store backend: postgres")
	concurrency := flag.Int("concurrency", envIntOrDefault("CONCURRENCY", 4), "number of concurrently running tasks")
	logLevel := flag.String("log-level", envOrDefault("LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	queueType := flag.String("queue", envOrDefault("QUEUE_TYPE", "redis"), "queue backend: redis")
	redisURL := flag.String("redis-url", envOrDefault("REDIS_URL", "redis://localhost:6379/0"), "redis connection URL")
	queueKey := flag.String("queue-key", envOrDefault("QUEUE_KEY", "dtq:tasks"), "redis list key for the task queue")
	flag.Parse()
	return Config{
		DatabaseURL:  stripDriverSuffix(*databaseURL),
		DBAutoCreate: *dbAutoCreate,
		StoreType:    *storeType,
		Concurrency:  *concurrency,
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

func stripDriverSuffix(url string) string {
	if i := strings.Index(url, "+"); i != -1 {
		if j := strings.Index(url[i:], "://"); j != -1 {
			return url[:i] + url[i+j:]
		}
	}
	return url
}
