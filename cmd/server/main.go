package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/middleware"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

func main() {
	// обработка аргументов командной строки
	cfg := parseConfig()

	// инициализация логера
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("init logger failure: %v", err)
	}
	defer logger.Log.Sync()

	// Создаём экземпляр хранилища
	metricsStorage := storage.NewMemStorage()
	metricsService := service.NewMetricsService(metricsStorage)

	r := chi.NewRouter()

	// пишем метрики
	r.Post("/update/{type}/{name}/{value}", handler.AcceptMetricsToStorage(metricsService))
	// смотрим метрики
	r.Get("/", handler.GetAllMetrics(metricsService))
	r.Get("/value/{type}/{name}", handler.GetMetric(metricsService))

	// активируем логирование запросов
	logger.Log.Info("Running server on: ", zap.String("address", cfg.RunAddr))
	if err := http.ListenAndServe(cfg.RunAddr, middleware.Conveyor(r, middleware.RequestLogger)); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
