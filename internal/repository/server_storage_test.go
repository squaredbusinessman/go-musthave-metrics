package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
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

func TestMemStorageUpdateMetricsBatch(t *testing.T) {
	ctx := context.Background()
	ms := NewMemStorage()

	gaugeValue := 12.5
	counterDelta := int64(3)
	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.MetricTypeGauge, Value: &gaugeValue},
		{ID: "PollCount", MType: models.MetricTypeCounter, Delta: &counterDelta},
	}

	if err := ms.UpdateMetricsBatch(ctx, metrics); err != nil {
		t.Fatalf("UpdateMetricsBatch() error = %v", err)
	}

	gauge, err := ms.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("GetGauge() error = %v", err)
	}
	if gauge != gaugeValue {
		t.Fatalf("gauge = %v, want %v", gauge, gaugeValue)
	}

	counter, err := ms.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("GetCounter() error = %v", err)
	}
	if counter != counterDelta {
		t.Fatalf("counter = %v, want %v", counter, counterDelta)
	}
}

func TestMemStorageUpdateMetricsBatchValidation(t *testing.T) {
	ctx := context.Background()
	ms := NewMemStorage()

	tests := []struct {
		name    string
		metrics []models.Metrics
		wantErr error
	}{
		{
			name:    "empty batch",
			metrics: nil,
		},
		{
			name: "missing gauge value",
			metrics: []models.Metrics{
				{ID: "Alloc", MType: models.MetricTypeGauge},
			},
			wantErr: apperr.ErrBadMetricValue,
		},
		{
			name: "missing counter delta",
			metrics: []models.Metrics{
				{ID: "PollCount", MType: models.MetricTypeCounter},
			},
			wantErr: apperr.ErrBadMetricValue,
		},
		{
			name: "unknown metric type",
			metrics: []models.Metrics{
				{ID: "Unknown", MType: "summary"},
			},
			wantErr: apperr.ErrUnknownMetricType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ms.UpdateMetricsBatch(ctx, tt.metrics)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateMetricsBatch() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
