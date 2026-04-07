package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

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

func buildRouter(h *handler.Handler, key string) http.Handler {
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

	return middleware.Conveyor(r, middleware.RequestLogger, middleware.HashMiddleware(key), middleware.GzipMiddleware)
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
	)

	switch {
	// Подключаемся к postgreSQL через драйвер pgx,
	// сразу используем пул в будущем эффективнее переиспользовать соединения
	// и распределять ресурсы
	case cfg.Database.DSN != "":
		pool, err := pgxpool.New(context.Background(), cfg.Database.DSN)
		if err != nil {
			log.Fatalf("db pool init failure: %v", err)
		}
		dbPool = pool
		defer dbPool.Close()

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

	var stopStore chan struct{}
	if fileStorage != nil && cfg.Storage.StoreInterval > 0 {
		stopStore = make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.Storage.StoreInterval) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := fileStorage.Save(); err != nil {
						myLog.Log.Error("periodic store failure", zap.Error(err))
					}
				case <-stopStore:
					return
				}
			}
		}()
	}

	router := buildRouter(h, cfg.Server.Key)

	// активируем логирование запросов
	myLog.Log.Info("Running server on: ", zap.String("address", cfg.Server.RunAddr))
	err := http.ListenAndServe(cfg.Server.RunAddr, router)
	if stopStore != nil {
		close(stopStore)
		if err := fileStorage.Save(); err != nil {
			myLog.Log.Error("final store failure", zap.Error(err))
		}
	}
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
