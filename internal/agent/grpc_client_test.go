package agent

import (
	"context"
	"net"
	"sync"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	pb "github.com/squaredbusinessman/go-musthave-metrics/internal/proto"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const grpcBufSize = 1024 * 1024

type testMetricsGRPCServer struct {
	pb.UnimplementedMetricsServer

	mu      sync.Mutex
	metrics []*pb.Metric
	realIP  string
}

func (s *testMetricsGRPCServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics = append([]*pb.Metric(nil), req.GetMetrics()...)
	if values := md.Get(metadataXRealIP); len(values) > 0 {
		s.realIP = values[0]
	}

	return &pb.UpdateMetricsResponse{}, nil
}

func newTestGRPCClient(t *testing.T, server pb.MetricsServer) pb.MetricsClient {
	t.Helper()

	listener := bufconn.Listen(grpcBufSize)
	grpcServer := grpc.NewServer()
	pb.RegisterMetricsServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return pb.NewMetricsClient(conn)
}

func TestSendMetricsBatchGRPC(t *testing.T) {
	server := &testMetricsGRPCServer{}
	client := newTestGRPCClient(t, server)

	value := 2.5
	delta := int64(3)
	err := SendMetricsBatchGRPC(context.Background(), client, []models.Metrics{
		{ID: "Alloc", MType: models.MetricTypeGauge, Value: &value},
		{ID: "PollCount", MType: models.MetricTypeCounter, Delta: &delta},
	})
	if err != nil {
		t.Fatalf("SendMetricsBatchGRPC() error = %v", err)
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.realIP == "" {
		t.Fatalf("%s metadata is empty", metadataXRealIP)
	}
	if len(server.metrics) != 2 {
		t.Fatalf("metrics len = %d, want 2", len(server.metrics))
	}

	if metric := server.metrics[0]; metric.GetId() != "Alloc" || metric.GetType() != pb.Metric_GAUGE || metric.GetValue() != 2.5 {
		t.Fatalf("gauge metric mismatch: %+v", metric)
	}
	if metric := server.metrics[1]; metric.GetId() != "PollCount" || metric.GetType() != pb.Metric_COUNTER || metric.GetDelta() != 3 {
		t.Fatalf("counter metric mismatch: %+v", metric)
	}
}

func TestReportMetricsGRPC(t *testing.T) {
	store := storage.NewMemStorage()
	ctx := context.Background()
	if err := store.SetGauge(ctx, "Alloc", models.Gauge{Value: 2.5}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := store.AddCounter(ctx, "PollCount", 3); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

	server := &testMetricsGRPCServer{}
	client := newTestGRPCClient(t, server)
	jobs := make(chan Job, 1)

	if err := ReportMetricsGRPC(ctx, client, store, jobs); err != nil {
		t.Fatalf("ReportMetricsGRPC() error = %v", err)
	}
	close(jobs)
	runJobs(jobs)

	server.mu.Lock()
	defer server.mu.Unlock()

	if len(server.metrics) != 2 {
		t.Fatalf("metrics len = %d, want 2", len(server.metrics))
	}
}
