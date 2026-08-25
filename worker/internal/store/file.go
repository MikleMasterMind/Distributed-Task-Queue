package store

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"distributed-task-queue/worker/internal/task"
)

type FileStore struct {
	dir    string
	logger *slog.Logger
	mu     sync.Mutex
}

func NewFileStore(dir string, logger *slog.Logger) *FileStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &FileStore{dir: dir, logger: logger}
}

func (s *FileStore) ListPending(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		t, err := s.readLocked(id)
		if err != nil {
			s.logger.Warn("skipping unreadable task file", "file", entry.Name(), "error", err)
			continue
		}
		if t.Status == task.StatusPending {
			ids = append(ids, t.ID)
		}
	}
	return ids, nil
}

func (s *FileStore) Claim(ctx context.Context, id string, startedAt time.Time) (task.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.readLocked(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return task.Task{}, ErrNotFound
		}
		return task.Task{}, err
	}
	if t.Status != task.StatusPending {
		return task.Task{}, ErrNotPending
	}
	t.Status = task.StatusRunning
	t.StartedAt = &startedAt
	if err := s.writeLocked(id, &t); err != nil {
		return task.Task{}, err
	}
	return t, nil
}

func (s *FileStore) Complete(ctx context.Context, id string, result map[string]any, finishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.readLocked(id)
	if err != nil {
		return err
	}
	t.Status = task.StatusSuccess
	t.Result = result
	t.Error = nil
	t.FinishedAt = &finishedAt
	return s.writeLocked(id, &t)
}

func (s *FileStore) Fail(ctx context.Context, id string, errMsg string, finishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.readLocked(id)
	if err != nil {
		return err
	}
	t.Status = task.StatusFailed
	t.Result = nil
	t.Error = &errMsg
	t.FinishedAt = &finishedAt
	return s.writeLocked(id, &t)
}

func (s *FileStore) readLocked(id string) (task.Task, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return task.Task{}, err
	}
	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return task.Task{}, err
	}
	s.logger.Debug("read task file", "file", s.path(id), "task_id", id, "status", t.Status)
	return t, nil
}

func (s *FileStore) writeLocked(id string, t *task.Task) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	path := s.path(id)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *FileStore) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}
