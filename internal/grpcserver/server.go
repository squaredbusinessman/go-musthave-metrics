package grpcserver

import (
	"context"
	"errors"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	pb "github.com/squaredbusinessman/go-musthave-metrics/internal/proto"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server реализует gRPC-сервис Metrics.
type Server struct {
	pb.UnimplementedMetricsServer

	metrics service.MetricsService
}

// New создаёт реализацию gRPC-сервиса Metrics.
func New(metrics service.MetricsService) *Server {
	return &Server{
		metrics: metrics,
	}
}

// UpdateMetrics принимает батч метрик и сохраняет его через общий слой сервиса.
func (s *Server) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	metrics, err := convertMetrics(req.GetMetrics())
	if err != nil {
		return nil, err
	}

	if err = s.metrics.UpdateMetricsBatch(ctx, metrics); err != nil {
		return nil, serviceErrorStatus(err)
	}

	return &pb.UpdateMetricsResponse{}, nil
}

func convertMetrics(metrics []*pb.Metric) ([]models.Metrics, error) {
	converted := make([]models.Metrics, 0, len(metrics))

	for _, metric := range metrics {
		if metric == nil {
			return nil, status.Error(codes.InvalidArgument, "metric is required")
		}
		if metric.GetId() == "" {
			return nil, status.Error(codes.InvalidArgument, "metric id is required")
		}

		switch metric.GetType() {
		case pb.Metric_GAUGE:
			value := metric.GetValue()
			converted = append(converted, models.Metrics{
				ID:    metric.GetId(),
				MType: models.MetricTypeGauge,
				Value: &value,
			})
		case pb.Metric_COUNTER:
			delta := metric.GetDelta()
			converted = append(converted, models.Metrics{
				ID:    metric.GetId(),
				MType: models.MetricTypeCounter,
				Delta: &delta,
			})
		default:
			return nil, status.Error(codes.InvalidArgument, "unknown metric type")
		}
	}

	return converted, nil
}

func serviceErrorStatus(err error) error {
	switch {
	case errors.Is(err, apperr.ErrBadMetricValue), errors.Is(err, apperr.ErrUnknownMetricType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
