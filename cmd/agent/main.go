package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func main() {
	// Инициализация логгера в агенте
	if err := myLog.Initialize("info"); err != nil {
		log.Fatalf("init logger failure: %v", err)
	}
	defer myLog.Log.Sync()

	cfg := parseConfig()

	store := storage.NewMemStorage()
	randS := rand.New(rand.NewSource(time.Now().UnixNano()))

	pollDuration := time.Duration(cfg.PollInterval) * time.Second
	pollTicker := time.NewTicker(pollDuration)
	defer pollTicker.Stop()

	reportDuration := time.Duration(cfg.ReportInterval) * time.Second
	reportTicker := time.NewTicker(reportDuration)
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
			agent.ReportMetrics(client, store, cfg.Addr, cfg.ReportFormat)
		}
	}
}
