package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultTasksDirFromWorkerDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(wd) != "worker" {
		t.Skip("test must run from the worker directory")
	}
	want := filepath.Join(filepath.Dir(wd), "data", "tasks")
	if got := defaultTasksDir(); got != want {
		t.Errorf("defaultTasksDir() = %s, want %s", got, want)
	}
}