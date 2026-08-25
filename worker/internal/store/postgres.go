package store

import (
	"context"
	"log/slog"
	"time"

	"distributed-task-queue/worker/internal/task"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TaskModel struct {
	ID         string         `gorm:"column:id;type:uuid;primaryKey"`
	Type       string         `gorm:"column:type;type:varchar(20);not null"`
	Payload    map[string]any `gorm:"column:payload;type:jsonb;not null"`
	Status     string         `gorm:"column:status;type:varchar(20);not null;default:PENDING"`
	Result     map[string]any `gorm:"column:result;type:jsonb"`
	Error      *string        `gorm:"column:error;type:text"`
	CreatedAt  time.Time      `gorm:"column:created_at;type:timestamptz;not null"`
	StartedAt  *time.Time     `gorm:"column:started_at;type:timestamptz"`
	FinishedAt *time.Time     `gorm:"column:finished_at;type:timestamptz"`
}

func (TaskModel) TableName() string {
	return "tasks"
}

type PostgresStore struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewPostgresStore(databaseURL string, autoCreate bool, log *slog.Logger) (*PostgresStore, error) {
	if log == nil {
		log = slog.Default()
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if autoCreate {
		if err := db.AutoMigrate(&TaskModel{}); err != nil {
			return nil, err
		}
	}
	return &PostgresStore{db: db, logger: log}, nil
}

func (s *PostgresStore) ListPending(ctx context.Context) ([]string, error) {
	var ids []string
	result := s.db.WithContext(ctx).
		Model(&TaskModel{}).
		Where("status = ?", string(task.StatusPending)).
		Pluck("id", &ids)
	if result.Error != nil {
		return nil, result.Error
	}
	return ids, nil
}

func (s *PostgresStore) Claim(ctx context.Context, id string, startedAt time.Time) (task.Task, error) {
	var model TaskModel
	result := s.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, string(task.StatusPending)).
		First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			s.db.WithContext(ctx).
				Where("id = ?", id).
				First(&model)
			if model.ID == "" {
				return task.Task{}, ErrNotFound
			}
			return task.Task{}, ErrNotPending
		}
		return task.Task{}, result.Error
	}

	result = s.db.WithContext(ctx).
		Model(&TaskModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(task.StatusRunning),
			"started_at": startedAt,
		})
	if result.Error != nil {
		return task.Task{}, result.Error
	}

	return s.toTask(model.ID, model.Type, model.Payload, string(task.StatusRunning), model.Result, model.Error, model.CreatedAt, &startedAt, model.FinishedAt), nil
}

func (s *PostgresStore) Complete(ctx context.Context, id string, result map[string]any, finishedAt time.Time) error {
	res := s.db.WithContext(ctx).
		Model(&TaskModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      string(task.StatusSuccess),
			"result":      result,
			"error":       nil,
			"finished_at": finishedAt,
		})
	return res.Error
}

func (s *PostgresStore) Fail(ctx context.Context, id string, errMsg string, finishedAt time.Time) error {
	res := s.db.WithContext(ctx).
		Model(&TaskModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      string(task.StatusFailed),
			"result":      nil,
			"error":       errMsg,
			"finished_at": finishedAt,
		})
	return res.Error
}

func (s *PostgresStore) toTask(id, typ string, payload map[string]any, status string, result map[string]any, errMsg *string, createdAt time.Time, startedAt, finishedAt *time.Time) task.Task {
	t := task.Task{
		ID:         id,
		Type:       typ,
		Payload:    payload,
		Status:     task.Status(status),
		Result:     result,
		Error:      errMsg,
		CreatedAt:  createdAt,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
	return t
}

func (s *PostgresStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *PostgresStore) DB() *gorm.DB {
	return s.db
}
