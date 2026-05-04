package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/cryptoutil"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/audit"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/buildinfo"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/middleware"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
	"github.com/squaredbusinessman/go-musthave-metrics/migrations"
	"go.uber.org/zap"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

const (
	shutdownTimeout         = 15 * time.Second
	serverReadHeaderTimeout = 2 * time.Second
	serverReadTimeout       = 5 * time.Second
	serverWriteTimeout      = 10 * time.Second
	serverIdleTimeout       = 60 * time.Second
)

func buildRouter(h *handler.Handler, key string, privateKey *rsa.PrivateKey, trustedSubnet string) http.Handler {
	r := chi.NewRouter()
	r.Use(chiMiddleware.StripSlashes)

	// пишем метрики
	r.Post("/update/{type}/{name}/{value}", h.AcceptMetricsToStorage)
	// новый эндпоинт для фиксации данных приходящих как JSON
	r.Post("/update", h.UpdateMetricJSON)
	// батч-обновление метрик
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/updates/", h.UpdateMetricsBatch)
	// смотрим метрики
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{type}/{name}", h.GetMetric)
	// получаем JSON со значением метрики из бд
	r.Post("/value", h.GetMetricJSON)
	r.Post("/value/", h.GetMetricJSON)
	// проверка соединения с БД
	r.Get("/ping", h.Ping)

	// не забыть что конвейер работает с миддлварами в обратном порядке
	return middleware.Conveyor(
		r,
		middleware.RequestLogger,
		middleware.HashMiddleware(key),
		middleware.GzipMiddleware,
		middleware.CryptoMiddleware(privateKey),
		middleware.TrustedSubnetMiddleware(trustedSubnet),
	)
}

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	// обработка аргументов командной строки
	cfg := parseConfig()

	// инициализация логера
	if err := myLog.Initialize(cfg.Server.LogLevel); err != nil {
		log.Fatalf("init logger failure: %v", err)
	}
	defer myLog.Log.Sync()

	var (
		store       repository.Storage
		fileStorage *repository.FileStorage
		dbPool      *pgxpool.Pool
		privateKey  *rsa.PrivateKey
	)

	if cfg.Crypto.KeyPath != "" {
		var err error
		privateKey, err = cryptoutil.LoadPrivateKey(cfg.Crypto.KeyPath)
		if err != nil {
			log.Fatalf("load private key failure: %v", err)
		}
	}

	switch {
	// Подключаемся к postgreSQL через драйвер pgx,
	// сразу используем пул в будущем эффективнее переиспользовать соединения
	// и распределять ресурсы
	case cfg.Database.DSN != "":
		pool, err := newDBPool(context.Background(), cfg.Database.DSN)
		if err != nil {
			log.Fatalf("db pool init failure: %v", err)
		}
		dbPool = pool

		if err = migrations.Up(dbPool, "migrations"); err != nil {
			myLog.Log.Error("migrations failure", zap.Error(err))
		}

		store = repository.NewDBStorage(dbPool)

	case cfg.Storage.FileStorageEnabled:
		store = repository.NewMemStorage()
		fileStorage = repository.NewFileStorage(cfg.Storage.FileStoragePath, store)

		if cfg.Storage.Restore {
			if err := fileStorage.Restore(); err != nil {
				log.Fatalf("restore metrics failure: %v", err)
			}
		}

	default:
		store = repository.NewMemStorage()
	}

	var serviceOpts []service.MetricsServiceOption
	if fileStorage != nil && cfg.Storage.StoreInterval == 0 {
		serviceOpts = append(serviceOpts, service.WithAfterUpdate(func() {
			if err := fileStorage.Save(); err != nil {
				myLog.Log.Error("synchronous store failure", zap.Error(err))
			}
		}))
	}

	metricsService := service.NewMetricsService(store, serviceOpts...)

	var auditObservers []audit.Observer
	// Аудит собирается как набор независимых приёмников.
	// Это даёт возможность писать сразу и в файл, и во внешний HTTP endpoint.
	if cfg.Audit.FilePath != "" {
		auditObservers = append(auditObservers, audit.NewFileObserver(cfg.Audit.FilePath))
	}
	if cfg.Audit.URL != "" {
		httpObserver, err := audit.NewHTTPObserver(cfg.Audit.URL, nil)
		if err != nil {
			log.Fatalf("audit http observer init failure: %v", err)
		}
		auditObservers = append(auditObservers, httpObserver)
	}

	var auditNotifier audit.Notifier
	if len(auditObservers) > 0 {
		auditNotifier = audit.NewPublisher(auditObservers...)
	}

	// Хендлер знает HTTP-контекст запроса, поэтому именно там удобно собирать событие аудита.
	h := handler.New(metricsService, dbPool, auditNotifier)

	var (
		stopStore context.CancelFunc
		storeWG   sync.WaitGroup
	)
	if fileStorage != nil && cfg.Storage.StoreInterval > 0 {
		storeCtx, cancelStore := context.WithCancel(context.Background())
		stopStore = cancelStore
		storeWG.Add(1)
		go func() {
			defer storeWG.Done()
			ticker := time.NewTicker(time.Duration(cfg.Storage.StoreInterval) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := fileStorage.Save(); err != nil {
						myLog.Log.Error("periodic store failure", zap.Error(err))
					}
				case <-storeCtx.Done():
					return
				}
			}
		}()
	}

	router := buildRouter(h, cfg.Server.Key, privateKey, cfg.Server.TrustedSubnet)
	server := &http.Server{
		Addr:              cfg.Server.RunAddr,
		Handler:           router,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
	serverErr := make(chan error, 1)
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stopSignals()

	// активируем логирование запросов
	myLog.Log.Info("Running server on: ", zap.String("address", cfg.Server.RunAddr))
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("could not start server: %v", err)
		}
	case <-signalCtx.Done():
		myLog.Log.Info("shutdown signal received")

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancelShutdown()

		if err := server.Shutdown(shutdownCtx); err != nil {
			myLog.Log.Error("server shutdown failure", zap.Error(err))
			if closeErr := server.Close(); closeErr != nil {
				myLog.Log.Error("server close failure", zap.Error(closeErr))
			}
		}

		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			myLog.Log.Error("server stopped with error", zap.Error(err))
		}
	}

	if stopStore != nil {
		stopStore()
		storeWG.Wait()
		if err := fileStorage.Save(); err != nil {
			myLog.Log.Error("final store failure", zap.Error(err))
		}
	}

	if dbPool != nil {
		dbPool.Close()
	}
}
