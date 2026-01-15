package repository

import (
	"context"
	"errors"
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
	ctx := context.Background()
	storage := NewMemStorage()

	if err := storage.SetGauge(ctx, "Alloc", models.Gauge{Value: 10}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if got := storage.gauges["Alloc"].Value; got != 10 {
		t.Fatalf("SetGauge stored %v, want 10", got)
	}

	if err := storage.AddCounter(ctx, "PollCount", 1); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}
	if err := storage.AddCounter(ctx, "PollCount", 2); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}
	if got := storage.counters["PollCount"].Value; got != 3 {
		t.Fatalf("AddCounter aggregated %v, want 3", got)
	}
}

func TestMemStorageSnapshotCopiesMaps(t *testing.T) {
	ctx := context.Background()
	storage := NewMemStorage()
	if err := storage.SetGauge(ctx, "RandomValue", models.Gauge{Value: 1}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := storage.AddCounter(ctx, "PollCount", 1); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

	gauges, counters, err := storage.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if err := storage.SetGauge(ctx, "RandomValue", models.Gauge{Value: 2}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := storage.AddCounter(ctx, "PollCount", 10); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

	if gauges["RandomValue"].Value != 1 {
		t.Fatalf("snapshot gauge changed, got %v want 1", gauges["RandomValue"].Value)
	}

	if counters["PollCount"].Value != 1 {
		t.Fatalf("snapshot counter changed, got %v want 1", counters["PollCount"].Value)
	}
}

func TestMemStorage_GetGauge(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		prepare func(*testing.T, *MemStorage)
		query   string
		want    float64
		wantErr error
	}{
		{
			name: "existing gauge",
			prepare: func(t *testing.T, ms *MemStorage) {
				if err := ms.SetGauge(ctx, "Alloc", models.Gauge{Value: 42}); err != nil {
					t.Fatalf("SetGauge() error = %v", err)
				}
			},
			query: "Alloc",
			want:  42,
		},
		{
			name:    "missing gauge",
			query:   "unknown",
			want:    0,
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStorage()
			if tt.prepare != nil {
				tt.prepare(t, ms)
			}
			got, err := ms.GetGauge(ctx, tt.query)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetGauge() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetGauge() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("GetGauge() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_GetCounter(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		prepare func(*testing.T, *MemStorage)
		query   string
		want    int64
		wantErr error
	}{
		{
			name: "existing counter",
			prepare: func(t *testing.T, ms *MemStorage) {
				if err := ms.AddCounter(ctx, "PollCount", 5); err != nil {
					t.Fatalf("AddCounter() error = %v", err)
				}
			},
			query: "PollCount",
			want:  5,
		},
		{
			name:    "missing counter",
			query:   "unknown",
			want:    0,
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStorage()
			if tt.prepare != nil {
				tt.prepare(t, ms)
			}
			got, err := ms.GetCounter(ctx, tt.query)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetCounter() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetCounter() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("GetCounter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
