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

	"github.com/go-resty/resty/v2"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/buildinfo"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/cryptoutil"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	pb "github.com/squaredbusinessman/go-musthave-metrics/internal/proto"
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

	var grpcClient pb.MetricsClient
	var closeGRPC func() error
	if cfg.GRPCAddr != "" {
		conn, metricsClient, err := agent.NewGRPCClient(cfg.GRPCAddr)
		if err != nil {
			log.Fatalf("init gRPC client failure: %v", err)
		}
		grpcClient = metricsClient
		closeGRPC = conn.Close
		defer func() {
			if err := closeGRPC(); err != nil {
				myLog.Log.Warn("gRPC client close failure", zap.Error(err))
			}
		}()
	}

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
				if err := reportMetrics(runCtx, cfg, client, grpcClient, store, publicKey, jobs); err != nil && runCtx.Err() == nil {
					myLog.Log.Error("report metrics failure", zap.Error(err))
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

	if err := reportMetrics(shutdownCtx, cfg, client, grpcClient, store, publicKey, jobs); err != nil && !logContextDone(err) {
		myLog.Log.Warn("final report metrics failure", zap.Error(err))
	}

	workerPool.Close()
	if err := workerPool.Wait(shutdownCtx); err != nil {
		myLog.Log.Warn("worker pool shutdown failure", zap.Error(err))
	}
}

func reportMetrics(
	ctx context.Context,
	cfg Config,
	httpClient *resty.Client,
	grpcClient pb.MetricsClient,
	store *storage.MemStorage,
	publicKey *rsa.PublicKey,
	jobs chan<- agent.Job,
) error {
	if cfg.GRPCAddr != "" {
		return agent.ReportMetricsGRPC(ctx, grpcClient, store, jobs)
	}

	return agent.ReportMetrics(ctx, httpClient, store, cfg.ReportFormat, cfg.Key, publicKey, jobs)
}

func logContextDone(err error) bool {
	return err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
