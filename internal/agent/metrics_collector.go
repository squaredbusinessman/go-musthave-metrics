package agent

import (
	"math/rand/v2"
	"runtime"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// collectRuntimeMetrics Функция сбора метрик агентского пакета
func collectRuntimeMetrics(store *storage.MemStorage, rnd *rand.Rand) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	store.SetGauge("Alloc", models.Gauge{Value: float64(stats.Alloc)})
	store.SetGauge("BuckHashSys", models.Gauge{Value: float64(stats.BuckHashSys)})
	store.SetGauge("Frees", models.Gauge{Value: float64(stats.Frees)})
	store.SetGauge("GCCPUFraction", models.Gauge{Value: stats.GCCPUFraction})
	store.SetGauge("GCSys", models.Gauge{Value: float64(stats.GCSys)})
	store.SetGauge("HeapAlloc", models.Gauge{Value: float64(stats.HeapAlloc)})
	store.SetGauge("HeapIdle", models.Gauge{Value: float64(stats.HeapIdle)})
	store.SetGauge("HeapInuse", models.Gauge{Value: float64(stats.HeapInuse)})
	store.SetGauge("HeapObjects", models.Gauge{Value: float64(stats.HeapObjects)})
	store.SetGauge("HeapReleased", models.Gauge{Value: float64(stats.HeapReleased)})
	store.SetGauge("HeapSys", models.Gauge{Value: float64(stats.HeapSys)})
	store.SetGauge("LastGC", models.Gauge{Value: float64(stats.LastGC)})
	store.SetGauge("Lookups", models.Gauge{Value: float64(stats.Lookups)})
	store.SetGauge("MCacheInuse", models.Gauge{Value: float64(stats.MCacheInuse)})
	store.SetGauge("MCacheSys", models.Gauge{Value: float64(stats.MCacheSys)})
	store.SetGauge("MSpanInuse", models.Gauge{Value: float64(stats.MSpanInuse)})
	store.SetGauge("MSpanSys", models.Gauge{Value: float64(stats.MSpanSys)})
	store.SetGauge("Mallocs", models.Gauge{Value: float64(stats.Mallocs)})
	store.SetGauge("NextGC", models.Gauge{Value: float64(stats.NextGC)})
	store.SetGauge("NumForcedGC", models.Gauge{Value: float64(stats.NumForcedGC)})
	store.SetGauge("NumGC", models.Gauge{Value: float64(stats.NumGC)})
	store.SetGauge("OtherSys", models.Gauge{Value: float64(stats.OtherSys)})
	store.SetGauge("PauseTotalNs", models.Gauge{Value: float64(stats.PauseTotalNs)})
	store.SetGauge("StackInuse", models.Gauge{Value: float64(stats.StackInuse)})
	store.SetGauge("StackSys", models.Gauge{Value: float64(stats.StackSys)})
	store.SetGauge("Sys", models.Gauge{Value: float64(stats.Sys)})
	store.SetGauge("TotalAlloc", models.Gauge{Value: float64(stats.TotalAlloc)})
	store.SetGauge("RandomValue", models.Gauge{Value: rnd.Float64()})

	store.SetCounter("PollCount", models.Counter{Value: 1})
}
