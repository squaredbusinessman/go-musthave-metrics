package main

import (
	"log"
	"net/http"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func main() {
	// Создаём экземпляр хранилища
	metricsStorage := storage.NewMemStorage()

	// Регистрируем обработчик из пакета handler с передачей хранилища
	http.HandleFunc("/update/", handler.AcceptMetricsToStorage(metricsStorage))

	log.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
