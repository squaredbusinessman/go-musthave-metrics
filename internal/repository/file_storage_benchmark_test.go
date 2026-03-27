package repository

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

const benchmarkMetricCount = 5000

func populateBenchmarkStorage(b *testing.B, store *MemStorage, metricCount int) {
	b.Helper()

	ctx := context.Background()
	for i := 0; i < metricCount; i++ {
		suffix := strconv.Itoa(i)
		if err := store.SetGauge(ctx, "gauge_"+suffix, models.Gauge{Value: float64(i) * 1.5}); err != nil {
			b.Fatalf("SetGauge() error = %v", err)
		}
		if err := store.AddCounter(ctx, "counter_"+suffix, int64(i)); err != nil {
			b.Fatalf("AddCounter() error = %v", err)
		}
	}
}

func BenchmarkFileStorageSave(b *testing.B) {
	store := NewMemStorage()
	populateBenchmarkStorage(b, store, benchmarkMetricCount)

	path := filepath.Join(b.TempDir(), "metrics.json")
	fs := NewFileStorage(path, store)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := fs.Save(); err != nil {
			b.Fatalf("Save() error = %v", err)
		}
	}
}

func BenchmarkFileStorageRestore(b *testing.B) {
	source := NewMemStorage()
	populateBenchmarkStorage(b, source, benchmarkMetricCount)

	path := filepath.Join(b.TempDir(), "metrics.json")
	if err := NewFileStorage(path, source).Save(); err != nil {
		b.Fatalf("Save() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		target := NewMemStorage()
		if err := NewFileStorage(path, target).Restore(); err != nil {
			b.Fatalf("Restore() error = %v", err)
		}
	}
}

func BenchmarkMemStorageUpdateMetricsBatch(b *testing.B) {
	metrics := make([]models.Metrics, 0, benchmarkMetricCount*2)
	for i := 0; i < benchmarkMetricCount; i++ {
		value := float64(i) * 1.5
		delta := int64(i)
		suffix := strconv.Itoa(i)

		metrics = append(metrics, models.Metrics{
			ID:    "gauge_" + suffix,
			MType: models.MetricTypeGauge,
			Value: &value,
		})
		metrics = append(metrics, models.Metrics{
			ID:    "counter_" + suffix,
			MType: models.MetricTypeCounter,
			Delta: &delta,
		})
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store := NewMemStorage()
		if err := store.UpdateMetricsBatch(context.Background(), metrics); err != nil {
			b.Fatalf("UpdateMetricsBatch() error = %v", err)
		}
	}
}
