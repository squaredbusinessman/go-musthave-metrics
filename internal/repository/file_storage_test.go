package repository

import (
	"path/filepath"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

func TestFileStorageSaveAndRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	source := NewMemStorage()
	source.SetGauge("Alloc", models.Gauge{Value: 100})
	source.AddCounter("PollCount", 5)

	if err := NewFileStorage(path, source).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	target := NewMemStorage()
	if err := NewFileStorage(path, target).Restore(); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	if got, ok := target.GetGauge("Alloc"); !ok || got != 100 {
		t.Fatalf("Restore() gauge = %v(ok=%v), want 100, true", got, ok)
	}

	if got, ok := target.GetCounter("PollCount"); !ok || got != 5 {
		t.Fatalf("Restore() counter = %v(ok=%v), want 5, true", got, ok)
	}
}

func TestFileStorageRestoreMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	storage := NewMemStorage()
	if err := NewFileStorage(path, storage).Restore(); err != nil {
		t.Fatalf("Restore() on missing file error = %v", err)
	}
}
