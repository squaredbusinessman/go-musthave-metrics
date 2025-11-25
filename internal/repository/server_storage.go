package repository

import (
	"sync"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

type Storage interface {
	SetGauge(name string, value models.Gauge)
	AddCounter(name string, value int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	Snapshot() (map[string]models.Gauge, map[string]models.Counter)
}

type MemStorage struct {
	mutex sync.RWMutex
	models.Gauge
	models.Counter
	gauges   map[string]models.Gauge
	counters map[string]models.Counter
}

// Snapshot - метод фиксации "снимка" карты метрик для передачи на сервер. Копируем значения мапы для потокобезопасности
func (ms *MemStorage) Snapshot() (map[string]models.Gauge, map[string]models.Counter) {
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

	return gaugesCopy, countersCopy
}

func (ms *MemStorage) GetGauge(name string) (float64, bool) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	g, ok := ms.gauges[name]
	return g.Value, ok
}

func (ms *MemStorage) GetCounter(name string) (int64, bool) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	c, ok := ms.counters[name]
	if !ok {
		return -1, false
	}
	return c.Value, ok
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]models.Gauge),
		counters: make(map[string]models.Counter),
	}
}

// SetGauge Фиксация изменения конкретной метрики
func (ms *MemStorage) SetGauge(name string, value models.Gauge) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.gauges[name] = value
}

// AddCounter Устанавливает значение счетчика
func (ms *MemStorage) AddCounter(name string, value int64) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	counter := ms.counters[name]
	counter.Value += value
	ms.counters[name] = counter
}
