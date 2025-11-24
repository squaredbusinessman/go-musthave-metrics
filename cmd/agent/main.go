package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	defaultPollInterval   = 2 * time.Second
	defaultReportInterval = 10 * time.Second
	defaultServerAddr     = "http://localhost:8080"
)

type gauge float64
type counter int64

// metricsStorage копит текущие значения метрик из runtime.
type metricsStorage struct {
	mu       sync.RWMutex
	gauges   map[string]gauge
	counters map[string]counter
}

func newMetricsStorage() *metricsStorage {
	return &metricsStorage{
		gauges:   make(map[string]gauge),
		counters: make(map[string]counter),
	}
}

func (s *metricsStorage) SetGauge(name string, value gauge) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *metricsStorage) AddCounter(name string, delta counter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += delta
}

func (s *metricsStorage) Snapshot() (map[string]gauge, map[string]counter) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gaugesCopy := make(map[string]gauge, len(s.gauges))
	for k, v := range s.gauges {
		gaugesCopy[k] = v
	}

	countersCopy := make(map[string]counter, len(s.counters))
	for k, v := range s.counters {
		countersCopy[k] = v
	}

	return gaugesCopy, countersCopy
}

func main() {
	store := newMetricsStorage()
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	pollTicker := time.NewTicker(defaultPollInterval)
	reportTicker := time.NewTicker(defaultReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	// Собрать стартовое состояние, чтобы отчёт не был пустым
	collectRuntimeMetrics(store, randSource)

	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-pollTicker.C:
			collectRuntimeMetrics(store, randSource)
		case <-reportTicker.C:
			reportMetrics(client, store, defaultServerAddr)
		}
	}
}

func collectRuntimeMetrics(store *metricsStorage, rnd *rand.Rand) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	store.SetGauge("Alloc", gauge(stats.Alloc))
	store.SetGauge("BuckHashSys", gauge(stats.BuckHashSys))
	store.SetGauge("Frees", gauge(stats.Frees))
	store.SetGauge("GCCPUFraction", gauge(stats.GCCPUFraction))
	store.SetGauge("GCSys", gauge(stats.GCSys))
	store.SetGauge("HeapAlloc", gauge(stats.HeapAlloc))
	store.SetGauge("HeapIdle", gauge(stats.HeapIdle))
	store.SetGauge("HeapInuse", gauge(stats.HeapInuse))
	store.SetGauge("HeapObjects", gauge(stats.HeapObjects))
	store.SetGauge("HeapReleased", gauge(stats.HeapReleased))
	store.SetGauge("HeapSys", gauge(stats.HeapSys))
	store.SetGauge("LastGC", gauge(stats.LastGC))
	store.SetGauge("Lookups", gauge(stats.Lookups))
	store.SetGauge("MCacheInuse", gauge(stats.MCacheInuse))
	store.SetGauge("MCacheSys", gauge(stats.MCacheSys))
	store.SetGauge("MSpanInuse", gauge(stats.MSpanInuse))
	store.SetGauge("MSpanSys", gauge(stats.MSpanSys))
	store.SetGauge("Mallocs", gauge(stats.Mallocs))
	store.SetGauge("NextGC", gauge(stats.NextGC))
	store.SetGauge("NumForcedGC", gauge(stats.NumForcedGC))
	store.SetGauge("NumGC", gauge(stats.NumGC))
	store.SetGauge("OtherSys", gauge(stats.OtherSys))
	store.SetGauge("PauseTotalNs", gauge(stats.PauseTotalNs))
	store.SetGauge("StackInuse", gauge(stats.StackInuse))
	store.SetGauge("StackSys", gauge(stats.StackSys))
	store.SetGauge("Sys", gauge(stats.Sys))
	store.SetGauge("TotalAlloc", gauge(stats.TotalAlloc))
	store.SetGauge("RandomValue", gauge(rnd.Float64()))

	store.AddCounter("PollCount", 1)
}

func reportMetrics(client *http.Client, store *metricsStorage, serverAddr string) {
	gauges, counters := store.Snapshot()
	for name, value := range gauges {
		if err := sendMetric(client, serverAddr, "gauge", name, strconv.FormatFloat(float64(value), 'f', -1, 64)); err != nil {
			log.Printf("send gauge %s: %v", name, err)
		}
	}

	for name, value := range counters {
		if err := sendMetric(client, serverAddr, "counter", name, strconv.FormatInt(int64(value), 10)); err != nil {
			log.Printf("send counter %s: %v", name, err)
		}
	}
}

func sendMetric(client *http.Client, serverAddr, metricType, name, value string) error {
	url := serverAddr + "/update/" + metricType + "/" + name + "/" + value
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	return nil
}
