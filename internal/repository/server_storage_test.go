package repository

import (
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

func TestNewMemStorage(t *testing.T) {
	storage := NewMemStorage()
	if storage == nil {
		t.Fatal("NewMemStorage returned nil")
	}
	if storage.gauges == nil || storage.counters == nil {
		t.Fatal("maps must be initialised in constructor")
	}
}

func TestMemStorageSetGaugeAndAddCounter(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGauge("Alloc", models.Gauge{Value: 10})
	if got := storage.gauges["Alloc"].Value; got != 10 {
		t.Fatalf("SetGauge stored %v, want 10", got)
	}

	storage.AddCounter("PollCount", 1)
	storage.AddCounter("PollCount", 2)
	if got := storage.counters["PollCount"].Value; got != 3 {
		t.Fatalf("AddCounter aggregated %v, want 3", got)
	}
}

func TestMemStorageSnapshotCopiesMaps(t *testing.T) {
	storage := NewMemStorage()
	storage.SetGauge("RandomValue", models.Gauge{Value: 1})
	storage.AddCounter("PollCount", 1)

	gauges, counters := storage.SnapShot()

	storage.SetGauge("RandomValue", models.Gauge{Value: 2})
	storage.AddCounter("PollCount", 10)

	if gauges["RandomValue"].Value != 1 {
		t.Fatalf("snapshot gauge changed, got %v want 1", gauges["RandomValue"].Value)
	}

	if counters["PollCount"].Value != 1 {
		t.Fatalf("snapshot counter changed, got %v want 1", counters["PollCount"].Value)
	}
}
