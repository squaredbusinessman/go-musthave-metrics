package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
)

func TestEncryptedAgentRequestThroughServerRouter(t *testing.T) {
	ctx := context.Background()
	hashKey := "secret"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	serverStore := repository.NewMemStorage()
	metricsService := service.NewMetricsService(serverStore)
	h := handler.New(metricsService, nil, nil)

	ts := httptest.NewServer(buildRouter(h, hashKey, privateKey))
	defer ts.Close()

	agentStore := repository.NewMemStorage()
	if err := agentStore.SetGauge(ctx, "Alloc", models.Gauge{Value: 42.5}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := agentStore.AddCounter(ctx, "PollCount", 3); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

	jobs := make(chan agent.Job, 10)
	client := resty.New().SetBaseURL(ts.URL)

	agent.ReportMetrics(client, agentStore, agent.ReportFormatPlain, hashKey, &privateKey.PublicKey, jobs)
	close(jobs)

	for job := range jobs {
		if err := job(); err != nil {
			t.Fatalf("job() error = %v", err)
		}
	}

	gauges, counters, err := serverStore.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if got := gauges["Alloc"].Value; got != 42.5 {
		t.Fatalf("Alloc = %v, want 42.5", got)
	}

	if got := counters["PollCount"].Value; got != 3 {
		t.Fatalf("PollCount = %d, want 3", got)
	}
}
