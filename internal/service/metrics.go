package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

type MetricsService interface {
	UpdateMetric(ctx context.Context, m models.Metric) error
	UpdateMetricJSON(ctx context.Context, m models.Metrics) error
	GetMetric(ctx context.Context, m models.Metric) (string, error)
	GetMetricJSON(ctx context.Context, m models.Metrics) (*models.Metrics, error)
	GetAllMetrics(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error)
}

type metricsService struct {
	store       repository.Storage
	afterUpdate func()
}

type MetricsServiceOption func(*metricsService)

func WithAfterUpdate(hook func()) MetricsServiceOption {
	return func(ms *metricsService) {
		ms.afterUpdate = hook
	}
}

func NewMetricsService(store repository.Storage, opts ...MetricsServiceOption) MetricsService {
	service := &metricsService{
		store: store,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

// UpdateMetric логика обновления метрик на сервере
func (s *metricsService) UpdateMetric(ctx context.Context, m models.Metric) error {
	switch m.Type {
	case models.MetricTypeGauge:
		val, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return apperr.ErrBadMetricValue
		}

		if err = s.store.SetGauge(ctx, m.Name, models.Gauge{Value: val}); err != nil {
			return err
		}

		s.triggerAfterUpdate()
		return nil

	case models.MetricTypeCounter:
		val, err := strconv.ParseInt(m.Value, 10, 64)
		if err != nil {
			return apperr.ErrBadMetricValue
		}

		if err = s.store.AddCounter(ctx, m.Name, val); err != nil {
			return err
		}

		s.triggerAfterUpdate()
		return nil

	default:
		return apperr.ErrUnknownMetricType
	}
}

func (s *metricsService) UpdateMetricJSON(ctx context.Context, m models.Metrics) error {
	switch m.MType {
	case models.MetricTypeGauge:
		if m.Value == nil {
			return apperr.ErrBadMetricValue
		}

		if err := s.store.SetGauge(ctx, m.ID, models.Gauge{Value: *m.Value}); err != nil {
			return err
		}
		s.triggerAfterUpdate()
		return nil
	case models.MetricTypeCounter:
		if m.Delta == nil {
			return apperr.ErrBadMetricValue
		}
		if err := s.store.AddCounter(ctx, m.ID, *m.Delta); err != nil {
			return err
		}
		s.triggerAfterUpdate()
		return nil
	default:
		return apperr.ErrUnknownMetricType
	}
}

func (s *metricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	for _, m := range metrics {
		if err := s.UpdateMetricJSON(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// GetMetric получение одной метрики, выводим строку для удобства использования в HTTP
func (s *metricsService) GetMetric(ctx context.Context, m models.Metric) (string, error) {
	switch m.Type {
	case models.MetricTypeGauge:
		g, err := s.store.GetGauge(ctx, m.Name)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return "", apperr.ErrBadMetricValue
			}
			return "", err
		}
		return strconv.FormatFloat(g, 'f', -1, 64), nil

	case models.MetricTypeCounter:
		c, err := s.store.GetCounter(ctx, m.Name)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return "", apperr.ErrMetricNotFound
			}
			return "", err
		}
		return strconv.FormatInt(c, 10), nil
	default:
		return "", apperr.ErrUnknownMetricType
	}
}

func (s *metricsService) GetMetricJSON(ctx context.Context, m models.Metrics) (*models.Metrics, error) {
	resp := models.Metrics{
		ID:    m.ID,
		MType: m.MType,
	}
	switch m.MType {
	case models.MetricTypeGauge:
		g, err := s.store.GetGauge(ctx, m.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperr.ErrMetricNotFound
			}
			return nil, err
		}
		resp.Value = &g

	case models.MetricTypeCounter:
		c, err := s.store.GetCounter(ctx, m.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperr.ErrMetricNotFound
			}
			return nil, err
		}
		resp.Delta = &c

	default:
		return nil, apperr.ErrUnknownMetricType
	}

	return &resp, nil
}

// GetAllMetrics снимок метрик зафиксированных в репозитории
func (s *metricsService) GetAllMetrics(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error) {
	g, c, err := s.store.Snapshot(ctx)
	if err != nil {
		return nil, nil, err
	}
	return g, c, nil
}

func (s *metricsService) triggerAfterUpdate() {
	if s.afterUpdate != nil {
		s.afterUpdate()
	}
}

// TODO: добавить сортировку, фильтрацию
