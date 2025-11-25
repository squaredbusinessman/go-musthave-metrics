package agent

import (
	"math/rand"
	"testing"

	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

var gaugeNames = []string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
	"MCacheSys",
	"MSpanInuse",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
	"RandomValue",
}

func TestCollectRuntimeMetricsPopulatesGauges(t *testing.T) {
	store := storage.NewMemStorage()
	seed := int64(42)
	rnd := rand.New(rand.NewSource(seed))
	expectedRnd := rand.New(rand.NewSource(seed))

	CollectRuntimeMetrics(store, rnd)

	gauges, counters := store.SnapShot()

	for _, name := range gaugeNames {
		if _, ok := gauges[name]; !ok {
			t.Fatalf("gauge %s not set", name)
		}
	}

	if got := counters["PollCount"].Value; got != 1 {
		t.Fatalf("PollCount = %d, want 1", got)
	}

	expectedRandom := expectedRnd.Float64()
	if got := gauges["RandomValue"].Value; got != expectedRandom {
		t.Fatalf("RandomValue = %v, want %v", got, expectedRandom)
	}
}

func TestCollectRuntimeMetricsIncrementsPollCount(t *testing.T) {
	store := storage.NewMemStorage()
	seed := int64(99)
	rnd := rand.New(rand.NewSource(seed))
	expectedRnd := rand.New(rand.NewSource(seed))

	CollectRuntimeMetrics(store, rnd)
	CollectRuntimeMetrics(store, rnd)

	gauges, counters := store.SnapShot()

	if got := counters["PollCount"].Value; got != 2 {
		t.Fatalf("PollCount = %d, want 2", got)
	}

	expectedRnd.Float64() // skip first draw
	expectedRandom := expectedRnd.Float64()
	if got := gauges["RandomValue"].Value; got != expectedRandom {
		t.Fatalf("RandomValue after second poll = %v, want %v", got, expectedRandom)
	}
}
