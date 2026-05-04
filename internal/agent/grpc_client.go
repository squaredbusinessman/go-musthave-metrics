package agent

import (
	"context"
	"errors"
	"strings"

	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	pb "github.com/squaredbusinessman/go-musthave-metrics/internal/proto"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const metadataXRealIP = "x-real-ip"

// NewGRPCClient создаёт gRPC-клиент для сервиса Metrics.
func NewGRPCClient(addr string) (*grpc.ClientConn, pb.MetricsClient, error) {
	conn, err := grpc.NewClient(normalizeGRPCAddr(addr), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return conn, pb.NewMetricsClient(conn), nil
}

// SendMetricsBatchGRPC отправляет батч метрик через gRPC.
func SendMetricsBatchGRPC(ctx context.Context, client pb.MetricsClient, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	if client == nil {
		return errors.New("gRPC metrics client must not be nil")
	}

	req := &pb.UpdateMetricsRequest{
		Metrics: metricsToProto(metrics),
	}

	return retry.Do(ctx, isRetryableGRPCErr, func() error {
		callCtx := metadata.AppendToOutgoingContext(ctx, metadataXRealIP, HostIP())
		_, err := client.UpdateMetrics(callCtx, req)
		return err
	})
}

// ReportMetricsGRPC ставит в очередь отправку всех накопленных метрик через gRPC.
func ReportMetricsGRPC(ctx context.Context, client pb.MetricsClient, store *storage.MemStorage, jobs chan<- Job) error {
	gauges, counters, err := store.Snapshot(ctx)
	if err != nil {
		myLog.Log.Warn("Failed to snapshot metrics", zap.Error(err))
		return err
	}

	metrics := snapshotToMetrics(gauges, counters)
	if len(metrics) == 0 {
		return nil
	}

	select {
	case jobs <- func(jobCtx context.Context) error {
		return SendMetricsBatchGRPC(jobCtx, client, metrics)
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func metricsToProto(metrics []models.Metrics) []*pb.Metric {
	result := make([]*pb.Metric, 0, len(metrics))

	for _, metric := range metrics {
		switch metric.MType {
		case models.MetricTypeGauge:
			if metric.Value == nil {
				continue
			}
			result = append(result, &pb.Metric{
				Id:    metric.ID,
				Type:  pb.Metric_GAUGE,
				Value: *metric.Value,
			})
		case models.MetricTypeCounter:
			if metric.Delta == nil {
				continue
			}
			result = append(result, &pb.Metric{
				Id:    metric.ID,
				Type:  pb.Metric_COUNTER,
				Delta: *metric.Delta,
			})
		}
	}

	return result
}

func normalizeGRPCAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

func isRetryableGRPCErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	switch status.Code(err) {
	case codes.Unavailable, codes.ResourceExhausted, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}
