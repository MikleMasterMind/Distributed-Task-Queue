package store

import (
	"fmt"
	"log/slog"

	"distributed-task-queue/worker/internal/config"
)

func NewFromConfig(cfg config.Config, logger *slog.Logger) (Store, error) {
	switch cfg.StoreType {
	case "postgres":
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required for postgres store")
		}
		return NewPostgresStore(cfg.DatabaseURL, cfg.DBAutoCreate, logger)
	default:
		return nil, fmt.Errorf("unknown store type %q", cfg.StoreType)
	}
}
