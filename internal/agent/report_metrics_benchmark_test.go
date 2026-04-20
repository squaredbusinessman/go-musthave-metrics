package agent

import (
	"context"
	"strconv"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

const benchmarkMetricCount = 5000

func benchmarkMetricStore(b *testing.B, metricCount int) *storage.MemStorage {
	b.Helper()

	store := storage.NewMemStorage()
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
	return store
}

func BenchmarkReportMetricsJSONScheduling(b *testing.B) {
	store := benchmarkMetricStore(b, benchmarkMetricCount)
	jobs := make(chan Job, 1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ReportMetrics(nil, store, ReportFormatJSON, "", nil, jobs)
		<-jobs
	}
}

func BenchmarkSnapshotToMetrics(b *testing.B) {
	store := benchmarkMetricStore(b, benchmarkMetricCount)
	gauges, counters, err := store.Snapshot(context.Background())
	if err != nil {
		b.Fatalf("Snapshot() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics := snapshotToMetrics(gauges, counters)
		if len(metrics) != benchmarkMetricCount*2 {
			b.Fatalf("snapshotToMetrics() len = %d, want %d", len(metrics), benchmarkMetricCount*2)
		}
	}
}
