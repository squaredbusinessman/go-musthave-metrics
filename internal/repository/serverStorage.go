package repository

import models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"

type Storage interface {
	SetGauge(name string, value models.Gauge)
	SetCounter(name string, value models.Counter)
}

type MemStorage struct {
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

func (ms *MemStorage) SetGauge(name string, value models.Gauge) {
	ms.gauges[name] = value
}

func (ms *MemStorage) SetCounter(name string, value models.Counter) {
	ms.counters[name] = value
}
