package service

import (
	"context"
	"errors"
	"testing"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func TestMetricsServiceUpdateAndGetMetric(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemStorage()
	svc := NewMetricsService(store)

	if err := svc.UpdateMetric(ctx, models.Metric{Type: models.MetricTypeGauge, Name: "Alloc", Value: "12.5"}); err != nil {
		t.Fatalf("UpdateMetric(gauge) error = %v", err)
	}
	if err := svc.UpdateMetric(ctx, models.Metric{Type: models.MetricTypeCounter, Name: "PollCount", Value: "5"}); err != nil {
		t.Fatalf("UpdateMetric(counter) error = %v", err)
	}
	if err := svc.UpdateMetric(ctx, models.Metric{Type: models.MetricTypeCounter, Name: "PollCount", Value: "2"}); err != nil {
		t.Fatalf("UpdateMetric(counter increment) error = %v", err)
	}

	gotGauge, err := svc.GetMetric(ctx, models.Metric{Type: models.MetricTypeGauge, Name: "Alloc"})
	if err != nil {
		t.Fatalf("GetMetric(gauge) error = %v", err)
	}
	if gotGauge != "12.5" {
		t.Fatalf("GetMetric(gauge) = %q, want %q", gotGauge, "12.5")
	}

	gotCounter, err := svc.GetMetric(ctx, models.Metric{Type: models.MetricTypeCounter, Name: "PollCount"})
	if err != nil {
		t.Fatalf("GetMetric(counter) error = %v", err)
	}
	if gotCounter != "7" {
		t.Fatalf("GetMetric(counter) = %q, want %q", gotCounter, "7")
	}
}

func TestMetricsServiceUpdateMetricErrors(t *testing.T) {
	ctx := context.Background()
	svc := NewMetricsService(repository.NewMemStorage())

	tests := []struct {
		name   string
		metric models.Metric
		want   error
	}{
		{
			name:   "bad gauge value",
			metric: models.Metric{Type: models.MetricTypeGauge, Name: "Alloc", Value: "abc"},
			want:   apperr.ErrBadMetricValue,
		},
		{
			name:   "bad counter value",
			metric: models.Metric{Type: models.MetricTypeCounter, Name: "PollCount", Value: "abc"},
			want:   apperr.ErrBadMetricValue,
		},
		{
			name:   "unknown type",
			metric: models.Metric{Type: "histogram", Name: "Custom", Value: "1"},
			want:   apperr.ErrUnknownMetricType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdateMetric(ctx, tt.metric)
			if !errors.Is(err, tt.want) {
				t.Fatalf("UpdateMetric() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestMetricsServiceJSONOperationsAndBatch(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemStorage()
	afterUpdateCalls := 0
	svc := NewMetricsService(store, WithAfterUpdate(func() {
		afterUpdateCalls++
	}))

	gaugeValue := 3.5
	if err := svc.UpdateMetricJSON(ctx, models.Metrics{
		ID:    "Alloc",
		MType: models.MetricTypeGauge,
		Value: &gaugeValue,
	}); err != nil {
		t.Fatalf("UpdateMetricJSON(gauge) error = %v", err)
	}

	counterDelta := int64(2)
	if err := svc.UpdateMetricJSON(ctx, models.Metrics{
		ID:    "PollCount",
		MType: models.MetricTypeCounter,
		Delta: &counterDelta,
	}); err != nil {
		t.Fatalf("UpdateMetricJSON(counter) error = %v", err)
	}

	batchGauge := 9.25
	batchDelta := int64(5)
	if err := svc.UpdateMetricsBatch(ctx, []models.Metrics{
		{
			ID:    "RandomValue",
			MType: models.MetricTypeGauge,
			Value: &batchGauge,
		},
		{
			ID:    "PollCount",
			MType: models.MetricTypeCounter,
			Delta: &batchDelta,
		},
	}); err != nil {
		t.Fatalf("UpdateMetricsBatch() error = %v", err)
	}

	if err := svc.UpdateMetricsBatch(ctx, nil); err != nil {
		t.Fatalf("UpdateMetricsBatch(nil) error = %v", err)
	}

	if afterUpdateCalls != 3 {
		t.Fatalf("afterUpdateCalls = %d, want 3", afterUpdateCalls)
	}

	gotGauge, err := svc.GetMetricJSON(ctx, models.Metrics{ID: "Alloc", MType: models.MetricTypeGauge})
	if err != nil {
		t.Fatalf("GetMetricJSON(gauge) error = %v", err)
	}
	if gotGauge.Value == nil || *gotGauge.Value != gaugeValue {
		t.Fatalf("GetMetricJSON(gauge) value = %+v, want %v", gotGauge.Value, gaugeValue)
	}

	gotCounter, err := svc.GetMetricJSON(ctx, models.Metrics{ID: "PollCount", MType: models.MetricTypeCounter})
	if err != nil {
		t.Fatalf("GetMetricJSON(counter) error = %v", err)
	}
	if gotCounter.Delta == nil || *gotCounter.Delta != 7 {
		t.Fatalf("GetMetricJSON(counter) delta = %+v, want 7", gotCounter.Delta)
	}

	gauges, counters, err := svc.GetAllMetrics(ctx)
	if err != nil {
		t.Fatalf("GetAllMetrics() error = %v", err)
	}
	if len(gauges) != 2 || len(counters) != 1 {
		t.Fatalf("GetAllMetrics() lens = (%d, %d), want (2, 1)", len(gauges), len(counters))
	}
}

func TestMetricsServiceJSONErrors(t *testing.T) {
	ctx := context.Background()
	svc := NewMetricsService(repository.NewMemStorage())

	gaugeValue := 1.0
	counterDelta := int64(1)

	tests := []struct {
		name   string
		metric models.Metrics
		want   error
	}{
		{
			name:   "missing gauge value",
			metric: models.Metrics{ID: "Alloc", MType: models.MetricTypeGauge},
			want:   apperr.ErrBadMetricValue,
		},
		{
			name:   "missing counter delta",
			metric: models.Metrics{ID: "PollCount", MType: models.MetricTypeCounter},
			want:   apperr.ErrBadMetricValue,
		},
		{
			name:   "unknown json type",
			metric: models.Metrics{ID: "Custom", MType: "histogram", Value: &gaugeValue},
			want:   apperr.ErrUnknownMetricType,
		},
		{
			name:   "unknown get type",
			metric: models.Metrics{ID: "Custom", MType: "histogram", Delta: &counterDelta},
			want:   apperr.ErrUnknownMetricType,
		},
		{
			name:   "missing metric",
			metric: models.Metrics{ID: "Missing", MType: models.MetricTypeGauge},
			want:   apperr.ErrMetricNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch tt.name {
			case "unknown get type", "missing metric":
				_, err = svc.GetMetricJSON(ctx, tt.metric)
			default:
				err = svc.UpdateMetricJSON(ctx, tt.metric)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}

	_, err := svc.GetMetric(ctx, models.Metric{Type: models.MetricTypeCounter, Name: "Missing"})
	if !errors.Is(err, apperr.ErrMetricNotFound) {
		t.Fatalf("GetMetric() error = %v, want %v", err, apperr.ErrMetricNotFound)
	}
}
