package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/middleware"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

func main() {
	// обработка аргументов командной строки
	cfg := parseConfig()

	// инициализация логера
	if err := myLog.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("init logger failure: %v", err)
	}
	defer myLog.Log.Sync()

	// Создаём экземпляр хранилища
	metricsStorage := storage.NewMemStorage()
	fileStorage := storage.NewFileStorage(cfg.FileStoragePath, metricsStorage)

	if cfg.Restore {
		if err := fileStorage.Restore(); err != nil {
			log.Fatalf("restore metrics failure: %v", err)
		}
	}

	var serviceOpts []service.MetricsServiceOption
	if cfg.StoreInterval == 0 {
		serviceOpts = append(serviceOpts, service.WithAfterUpdate(func() {
			if err := fileStorage.Save(); err != nil {

				myLog.Log.Error("synchronous store failure", zap.Error(err))
			}
		}))
	}
	metricsService := service.NewMetricsService(metricsStorage, serviceOpts...)

	var stopStore chan struct{}
	if cfg.StoreInterval > 0 {
		stopStore = make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
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

	r := chi.NewRouter()
	r.Use(chiMiddleware.StripSlashes)

	// пишем метрики
	r.Post("/update/{type}/{name}/{value}", handler.AcceptMetricsToStorage(metricsService))
	// новый эндпоинт для фиксации данных приходящих как JSON
	r.Post("/update", handler.UpdateMetricJSON(metricsService))
	// смотрим метрики
	r.Get("/", handler.GetAllMetrics(metricsService))
	r.Get("/value/{type}/{name}", handler.GetMetric(metricsService))
	// получаем JSON со значением метрики из бд
	r.Post("/value", handler.GetMetricJSON(metricsService))

	// активируем логирование запросов
	myLog.Log.Info("Running server on: ", zap.String("address", cfg.RunAddr))
	err := http.ListenAndServe(cfg.RunAddr, middleware.Conveyor(r, middleware.RequestLogger, middleware.GzipMiddleware))
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
