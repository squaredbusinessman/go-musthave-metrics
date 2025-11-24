package repository

import (
	"sync"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

type Storage interface {
	SetGauge(name string, value models.Gauge)
	SetCounter(name string, value models.Counter)
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

// SetGauge Фиксация изменения конкретной метрики
func (ms *MemStorage) SetGauge(name string, value models.Gauge) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.gauges[name] = value
}

// SetCounter Устанавливает значение счетчика
func (ms *MemStorage) SetCounter(name string, value models.Counter) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.counters[name] = value
}

// SnapShot - метод фиксации "снимка" карты метрик для передачи на сервер. Копируем значения мапы для потокобезопасности
func (ms *MemStorage) SnapShot() (map[string]models.Gauge, map[string]models.Counter) {
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
