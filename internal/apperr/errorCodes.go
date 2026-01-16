package apperr

import "errors"

var (
	ErrUnknownMetricType = errors.New("unknown metric type")
	ErrBadMetricValue    = errors.New("bad metric value")
	ErrMetricNotFound    = errors.New("metric not found")
)
