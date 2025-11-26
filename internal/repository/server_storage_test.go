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

	gauges, counters := storage.Snapshot()

	storage.SetGauge("RandomValue", models.Gauge{Value: 2})
	storage.AddCounter("PollCount", 10)

	if gauges["RandomValue"].Value != 1 {
		t.Fatalf("snapshot gauge changed, got %v want 1", gauges["RandomValue"].Value)
	}

	if counters["PollCount"].Value != 1 {
		t.Fatalf("snapshot counter changed, got %v want 1", counters["PollCount"].Value)
	}
}

func TestMemStorage_GetGauge(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*MemStorage)
		query   string
		want    float64
		ok      bool
	}{
		{
			name: "existing gauge",
			prepare: func(ms *MemStorage) {
				ms.SetGauge("Alloc", models.Gauge{Value: 42})
			},
			query: "Alloc",
			want:  42,
			ok:    true,
		},
		{
			name:  "missing gauge",
			query: "unknown",
			want:  0,
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStorage()
			if tt.prepare != nil {
				tt.prepare(ms)
			}
			got, ok := ms.GetGauge(tt.query)
			if got != tt.want {
				t.Errorf("GetGauge() got = %v, want %v", got, tt.want)
			}
			if ok != tt.ok {
				t.Errorf("GetGauge() ok = %v, want %v", ok, tt.ok)
			}
		})
	}
}

func TestMemStorage_GetCounter(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*MemStorage)
		query   string
		want    int64
		ok      bool
	}{
		{
			name: "existing counter",
			prepare: func(ms *MemStorage) {
				ms.AddCounter("PollCount", 5)
			},
			query: "PollCount",
			want:  5,
			ok:    true,
		},
		{
			name:  "missing counter",
			query: "unknown",
			want:  0,
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStorage()
			if tt.prepare != nil {
				tt.prepare(ms)
			}
			got, ok := ms.GetCounter(tt.query)
			if got != tt.want {
				t.Errorf("GetCounter() got = %v, want %v", got, tt.want)
			}
			if ok != tt.ok {
				t.Errorf("GetCounter() ok = %v, want %v", ok, tt.ok)
			}
		})
	}
}
