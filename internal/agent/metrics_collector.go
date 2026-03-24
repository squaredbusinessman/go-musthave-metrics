package agent

import (
	"context"
	"math/rand"
	"runtime"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// CollectRuntimeMetrics - собирает runtime-метрики и обновляет PollCount.
func CollectRuntimeMetrics(store *storage.MemStorage, rnd *rand.Rand) {
	var stats runtime.MemStats
	ctx := context.Background()
	runtime.ReadMemStats(&stats)

	_ = store.SetGauge(ctx, "Alloc", models.Gauge{Value: float64(stats.Alloc)})
	_ = store.SetGauge(ctx, "BuckHashSys", models.Gauge{Value: float64(stats.BuckHashSys)})
	_ = store.SetGauge(ctx, "Frees", models.Gauge{Value: float64(stats.Frees)})
	_ = store.SetGauge(ctx, "GCCPUFraction", models.Gauge{Value: stats.GCCPUFraction})
	_ = store.SetGauge(ctx, "GCSys", models.Gauge{Value: float64(stats.GCSys)})
	_ = store.SetGauge(ctx, "HeapAlloc", models.Gauge{Value: float64(stats.HeapAlloc)})
	_ = store.SetGauge(ctx, "HeapIdle", models.Gauge{Value: float64(stats.HeapIdle)})
	_ = store.SetGauge(ctx, "HeapInuse", models.Gauge{Value: float64(stats.HeapInuse)})
	_ = store.SetGauge(ctx, "HeapObjects", models.Gauge{Value: float64(stats.HeapObjects)})
	_ = store.SetGauge(ctx, "HeapReleased", models.Gauge{Value: float64(stats.HeapReleased)})
	_ = store.SetGauge(ctx, "HeapSys", models.Gauge{Value: float64(stats.HeapSys)})
	_ = store.SetGauge(ctx, "LastGC", models.Gauge{Value: float64(stats.LastGC)})
	_ = store.SetGauge(ctx, "Lookups", models.Gauge{Value: float64(stats.Lookups)})
	_ = store.SetGauge(ctx, "MCacheInuse", models.Gauge{Value: float64(stats.MCacheInuse)})
	_ = store.SetGauge(ctx, "MCacheSys", models.Gauge{Value: float64(stats.MCacheSys)})
	_ = store.SetGauge(ctx, "MSpanInuse", models.Gauge{Value: float64(stats.MSpanInuse)})
	_ = store.SetGauge(ctx, "MSpanSys", models.Gauge{Value: float64(stats.MSpanSys)})
	_ = store.SetGauge(ctx, "Mallocs", models.Gauge{Value: float64(stats.Mallocs)})
	_ = store.SetGauge(ctx, "NextGC", models.Gauge{Value: float64(stats.NextGC)})
	_ = store.SetGauge(ctx, "NumForcedGC", models.Gauge{Value: float64(stats.NumForcedGC)})
	_ = store.SetGauge(ctx, "NumGC", models.Gauge{Value: float64(stats.NumGC)})
	_ = store.SetGauge(ctx, "OtherSys", models.Gauge{Value: float64(stats.OtherSys)})
	_ = store.SetGauge(ctx, "PauseTotalNs", models.Gauge{Value: float64(stats.PauseTotalNs)})
	_ = store.SetGauge(ctx, "StackInuse", models.Gauge{Value: float64(stats.StackInuse)})
	_ = store.SetGauge(ctx, "StackSys", models.Gauge{Value: float64(stats.StackSys)})
	_ = store.SetGauge(ctx, "Sys", models.Gauge{Value: float64(stats.Sys)})
	_ = store.SetGauge(ctx, "TotalAlloc", models.Gauge{Value: float64(stats.TotalAlloc)})
	_ = store.SetGauge(ctx, "RandomValue", models.Gauge{Value: rnd.Float64()})

	_ = store.AddCounter(ctx, "PollCount", 1)
}
