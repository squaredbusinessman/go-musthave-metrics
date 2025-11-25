package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func main() {
	// Создаём экземпляр хранилища
	metricsStorage := storage.NewMemStorage()

	r := chi.NewRouter()

	// Регистрируем обработчик из пакета handler с передачей хранилища
	r.Post("/update/", handler.AcceptMetricsToStorage(metricsStorage))

	log.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
