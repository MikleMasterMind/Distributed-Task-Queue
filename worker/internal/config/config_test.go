package config

import (
	"os"
	"testing"
)

func TestLoadReadsEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("STORE_TYPE", "postgres")
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
	if cfg.StoreType != "postgres" {
		t.Errorf("StoreType = %q, want %q", cfg.StoreType, "postgres")
	}
	if cfg.Concurrency != 8 {
		t.Errorf("Concurrency = %d, want 8", cfg.Concurrency)
	}
}

func TestStripDriverSuffix(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"postgresql+asyncpg://u:p@localhost/db", "postgresql://u:p@localhost/db"},
		{"postgresql://u:p@localhost/db", "postgresql://u:p@localhost/db"},
		{"postgres+pgx://u:p@localhost/db", "postgres://u:p@localhost/db"},
		{"sqlite:///tmp/db.sqlite", "sqlite:///tmp/db.sqlite"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := stripDriverSuffix(tt.in); got != tt.want {
			t.Errorf("stripDriverSuffix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
