package repository

import (
	"context"
	"path/filepath"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

func TestFileStorageSaveAndRestore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metrics.json")
	source := NewMemStorage()
	if err := source.SetGauge(ctx, "Alloc", models.Gauge{Value: 100}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := source.AddCounter(ctx, "PollCount", 5); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

	if err := NewFileStorage(path, source).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	target := NewMemStorage()
	if err := NewFileStorage(path, target).Restore(); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	if got, err := target.GetGauge(ctx, "Alloc"); err != nil || got != 100 {
		t.Fatalf("Restore() gauge = %v(err=%v), want 100, nil", got, err)
	}

	if got, err := target.GetCounter(ctx, "PollCount"); err != nil || got != 5 {
		t.Fatalf("Restore() counter = %v(err=%v), want 5, nil", got, err)
	}
}

func TestFileStorageRestoreMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	storage := NewMemStorage()
	if err := NewFileStorage(path, storage).Restore(); err != nil {
		t.Fatalf("Restore() on missing file error = %v", err)
	}
}
