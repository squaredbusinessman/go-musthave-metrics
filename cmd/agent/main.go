package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/buildinfo"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	// Инициализация логгера в агенте
	if err := myLog.Initialize("info"); err != nil {
		log.Fatalf("init logger failure: %v", err)
	}
	defer myLog.Log.Sync()

	cfg := parseConfig()

	jobs := make(chan agent.Job, cfg.RateLimit)
	agent.StartWorkers(cfg.RateLimit, jobs)

	store := storage.NewMemStorage()
	randS := rand.New(rand.NewSource(time.Now().UnixNano()))

	client := resty.New().
		SetBaseURL("http://" + cfg.Addr).
		SetTimeout(5 * time.Second)

	// горутина фиксации рантайм-метрик
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		agent.CollectRuntimeMetrics(store, randS)
		for range ticker.C {
			agent.CollectRuntimeMetrics(store, randS)
		}
	}()

	// горутина фиксация gopsutil метрик
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		agent.ColleсtGopsutilMetrics(store)
		for range ticker.C {
			agent.ColleсtGopsutilMetrics(store)
		}
	}()

	// горутина отправка метрик
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			agent.ReportMetrics(client, store, cfg.ReportFormat, cfg.Key, jobs)
		}
	}()

	// блокировка main
	select {}
}
