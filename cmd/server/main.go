package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func main() {
	// обработка аргументов командной строки
	cfg := parseConfig()

	// Создаём экземпляр хранилища
	metricsStorage := storage.NewMemStorage()

	r := chi.NewRouter()

	// пишем метрики
	r.Post("/update/{type}/{name}/{value}", handler.AcceptMetricsToStorage(metricsStorage))
	// смотрим метрики
	r.Get("/", handler.GetAllMetrics(metricsStorage))
	r.Get("/value/{type}/{name}", handler.GetMetric(metricsStorage))

	log.Println("Listening on port 8080", cfg.RunAddr)
	if err := http.ListenAndServe(cfg.RunAddr, r); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
