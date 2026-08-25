package config

import (
	"os"
	"testing"
)

func TestLoadReadsEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("STORE_TYPE", "file")
	os.Setenv("QUEUE_TYPE", "redis")
	os.Setenv("CONCURRENCY", "8")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("STORE_TYPE")
	defer os.Unsetenv("QUEUE_TYPE")
	defer os.Unsetenv("CONCURRENCY")

	os.Args = []string{"test"}
	cfg := Load()

	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.StoreType != "file" {
		t.Errorf("StoreType = %q, want %q", cfg.StoreType, "file")
	}
	if cfg.Concurrency != 8 {
		t.Errorf("Concurrency = %d, want 8", cfg.Concurrency)
	}
}
