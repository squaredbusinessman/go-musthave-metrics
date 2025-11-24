package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	serverAddr     = "localhost:8080"
)

func main() {
	store := storage.NewMemStorage()
	randS := rand.New(rand.NewSource(time.Now().UnixNano()))

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	agent.CollectRuntimeMetrics(store, randS)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for {
		select {
		case <-pollTicker.C:
			agent.CollectRuntimeMetrics(store, randS)
		case <-reportTicker.C:
			agent.ReportMetrics(client, store, serverAddr)
		}
	}
}
