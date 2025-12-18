package service

import (
	"context"
	"errors"
	"strconv"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

const (
	MetricTypeGauge   = "gauge"
	MetricTypeCounter = "counter"
)

var (
	ErrUnknownMetricType = errors.New("unknown metric type")
	ErrBadMetricValue    = errors.New("bad metric value")
	ErrMetricNotFound    = errors.New("metric not found")
)

type MetricsService interface {
	UpdateMetric(ctx context.Context, m models.Metric) error
	UpdateMetricJSON(ctx context.Context, m models.Metrics) error
	GetMetric(ctx context.Context, m models.Metric) (string, error)
	GetAllMetrics(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error)
}

type metricsService struct {
	store repository.Storage
}

func NewMetricsService(store repository.Storage) MetricsService {
	return &metricsService{
		store: store,
	}
}

// UpdateMetric логика обновления метрик на сервере
func (s *metricsService) UpdateMetric(ctx context.Context, m models.Metric) error {
	switch m.Type {
	case MetricTypeGauge:
		val, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return ErrBadMetricValue
		}
		s.store.SetGauge(m.Name, models.Gauge{Value: val})
		return nil

	case MetricTypeCounter:
		val, err := strconv.ParseInt(m.Value, 10, 64)
		if err != nil {
			return ErrBadMetricValue
		}
		s.store.AddCounter(m.Name, val)
		return nil

	default:
		return ErrUnknownMetricType
	}
}

func (s *metricsService) UpdateMetricJSON(ctx context.Context, m models.Metrics) error {
	switch m.MType {
	case MetricTypeGauge:
		if m.Value == nil {
			return ErrBadMetricValue
		}
		s.store.SetGauge(m.ID, models.Gauge{
			Value: *m.Value,
		})
		return nil
	case MetricTypeCounter:
		if m.Delta == nil {
			return ErrBadMetricValue
		}
		s.store.AddCounter(m.ID, *m.Delta)
		return nil
	default:
		return ErrUnknownMetricType
	}
}

// GetMetric получение одной метрики, выводим строку для удобства использования в HTTP
func (s *metricsService) GetMetric(ctx context.Context, m models.Metric) (string, error) {
	switch m.Type {
	case MetricTypeGauge:
		g, ok := s.store.GetGauge(m.Name)
		if !ok {
			return "", ErrMetricNotFound
		}
		return strconv.FormatFloat(g, 'f', -1, 64), nil

	case MetricTypeCounter:
		c, ok := s.store.GetCounter(m.Name)
		if !ok {
			return "", ErrMetricNotFound
		}
		return strconv.FormatInt(c, 10), nil

	default:
		return "", ErrUnknownMetricType
	}
}

// GetAllMetrics снимок метрик зафиксированных в репозитории
// TODO: добавить сортировку, фильтрацию
func (s *metricsService) GetAllMetrics(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error) {
	g, c := s.store.Snapshot()
	return g, c, nil
}
