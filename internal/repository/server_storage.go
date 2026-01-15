package repository

import (
	"context"
	"errors"
	"sync"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

var ErrNotFound = errors.New("metric not found")

type Storage interface {
	SetGauge(ctx context.Context, name string, value models.Gauge)
	AddCounter(ctx context.Context, name string, value int64)
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	Snapshot(ctx context.Context, ) (map[string]models.Gauge, map[string]models.Counter)
}

type MemStorage struct {
	mutex sync.RWMutex
	models.Gauge
	models.Counter
	gauges   map[string]models.Gauge
	counters map[string]models.Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]models.Gauge),
		counters: make(map[string]models.Counter),
	}
}

// Snapshot - метод фиксации "снимка" карты метрик для передачи на сервер. Копируем значения мапы для потокобезопасности
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

func (ms *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	g, ok := ms.gauges[name]
	return g.Value, ok, nil
}

func (ms *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	c, ok := ms.counters[name]
	if !ok {
		return 0, false, nil
	}
	return c.Value, ok, nil
}

// SetGauge Фиксация изменения конкретной метрики
func (ms *MemStorage) SetGauge(ctx context.Context, name string, value models.Gauge) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.gauges[name] = value
	return nil
}

// AddCounter Устанавливает значение счетчика
func (ms *MemStorage) AddCounter(ctx context.Context, name string, value int64) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	counter := ms.counters[name]
	counter.Value += value
	ms.counters[name] = counter
	return nil
}
