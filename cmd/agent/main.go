package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/buildinfo"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/cryptoutil"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"go.uber.org/zap"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

const shutdownTimeout = 15 * time.Second
const timeout = 5 * time.Second
const retryCount = 3
const waitTime = 1 * time.Second
const maxWaitTime = 5 * time.Second

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	// Инициализация логгера в агенте
	if err := myLog.Initialize("info"); err != nil {
		log.Fatalf("init logger failure: %v", err)
	}
	defer myLog.Log.Sync()

	cfg := parseConfig()

	var publicKey *rsa.PublicKey
	if cfg.CryptoKey != "" {
		var err error
		publicKey, err = cryptoutil.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			log.Fatalf("load crypto public key failure: %v", err)
		}

		cfg.ReportFormat = agent.ReportFormatJSON
	}

	workerPool := agent.NewWorkerPool(cfg.RateLimit, cfg.RateLimit)
	jobs := workerPool.Jobs()

	store := storage.NewMemStorage()
	randS := rand.New(rand.NewSource(time.Now().UnixNano()))

	client := agent.NewHTTPClient(cfg.Addr, timeout, retryCount, waitTime, maxWaitTime)

	defer client.GetClient().CloseIdleConnections()

	runCtx, stopRun := context.WithCancel(context.Background())
	defer stopRun()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stopSignals()

	var producers sync.WaitGroup

	// горутина фиксации рантайм-метрик
	producers.Add(1)
	go func() {
		defer producers.Done()
		ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		agent.CollectRuntimeMetrics(store, randS)
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				agent.CollectRuntimeMetrics(store, randS)
			}
		}
	}()

	// горутина фиксация gopsutil метрик
	producers.Add(1)
	go func() {
		defer producers.Done()
		ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		agent.ColleсtGopsutilMetrics(store)
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				agent.ColleсtGopsutilMetrics(store)
			}
		}
	}()

	// горутина отправка метрик
	producers.Add(1)
	go func() {
		defer producers.Done()
		ticker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if err := agent.ReportMetrics(runCtx, client, store, cfg.ReportFormat, cfg.Key, publicKey, jobs); err != nil && runCtx.Err() == nil {
					myLog.Log.Warn("report metrics failure", zap.Error(err))
				}
			}
		}
	}()

	<-signalCtx.Done()
	myLog.Log.Info("shutdown signal received")

	stopRun()
	producers.Wait()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := agent.ReportMetrics(shutdownCtx, client, store, cfg.ReportFormat, cfg.Key, publicKey, jobs); err != nil && !logContextDone(err) {
		myLog.Log.Warn("final report metrics failure", zap.Error(err))
	}

	workerPool.Close()
	if err := workerPool.Wait(shutdownCtx); err != nil {
		myLog.Log.Warn("worker pool shutdown failure", zap.Error(err))
	}
}

func logContextDone(err error) bool {
	return err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
