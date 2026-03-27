package apperr

import "errors"

var (
	// ErrUnknownMetricType - передан неизвестный тип метрики.
	ErrUnknownMetricType = errors.New("unknown metric type")
	// ErrBadMetricValue - передано некорректное значение метрики.
	ErrBadMetricValue = errors.New("bad metric value")
	// ErrMetricNotFound - запрошенная метрика отсутствует.
	ErrMetricNotFound = errors.New("metric not found")
)
