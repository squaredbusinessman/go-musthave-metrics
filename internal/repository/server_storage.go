package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

// ErrNotFound - метрика не найдена в хранилище.
var ErrNotFound = errors.New("metric not found")

// Storage - общий интерфейс хранилища метрик.
type Storage interface {
	SetGauge(ctx context.Context, name string, value models.Gauge) error
	AddCounter(ctx context.Context, name string, value int64) error
	GetGauge(ctx context.Context, name string) (float64, error)
	GetCounter(ctx context.Context, name string) (int64, error)
	Snapshot(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error)
	UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
}

// MemStorage - in-memory хранилище метрик.
type MemStorage struct {
	mutex sync.RWMutex
	models.Gauge
	models.Counter
	gauges   map[string]models.Gauge
	counters map[string]models.Counter
}

// NewMemStorage - создает пустое хранилище в памяти.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]models.Gauge),
		counters: make(map[string]models.Counter),
	}
}

// SetGauge - сохраняет значение gauge-метрики.
func (ms *MemStorage) SetGauge(ctx context.Context, name string, value models.Gauge) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.gauges[name] = value
	return nil
}

// AddCounter - увеличивает значение counter-метрики.
func (ms *MemStorage) AddCounter(ctx context.Context, name string, value int64) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	counter := ms.counters[name]
	counter.Value += value
	ms.counters[name] = counter
	return nil
}

// GetGauge - возвращает значение gauge-метрики.
func (ms *MemStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	g, ok := ms.gauges[name]
	if !ok {
		return 0, ErrNotFound
	}
	return g.Value, nil
}

// GetCounter - возвращает значение counter-метрики.
func (ms *MemStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	c, ok := ms.counters[name]
	if !ok {
		return 0, ErrNotFound
	}
	return c.Value, nil
}

// Snapshot - возвращает копию текущих метрик для безопасного чтения.
func (ms *MemStorage) Snapshot(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	gaugesCopy := make(map[string]models.Gauge, len(ms.gauges))
	for k, v := range ms.gauges {
		gaugesCopy[k] = v
	}

	countersCopy := make(map[string]models.Counter, len(ms.counters))
	for k, v := range ms.counters {
		countersCopy[k] = v
	}

	return gaugesCopy, countersCopy, nil
}

// UpdateMetricsBatch - обновляет набор метрик под одной блокировкой.
func (ms *MemStorage) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.MetricTypeGauge:
			if metric.Value == nil {
				return apperr.ErrBadMetricValue
			}
		case models.MetricTypeCounter:
			if metric.Delta == nil {
				return apperr.ErrBadMetricValue
			}
		default:
			return apperr.ErrUnknownMetricType
		}
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.MetricTypeGauge:
			ms.gauges[metric.ID] = models.Gauge{Value: *metric.Value}
		case models.MetricTypeCounter:
			counter := ms.counters[metric.ID]
			counter.Value += *metric.Delta
			ms.counters[metric.ID] = counter
		}
	}

	return nil
}
