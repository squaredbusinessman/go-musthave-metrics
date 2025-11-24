package agent

import (
	"log"
	"net/http"
	"strconv"

	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// ReportMetrics функция отправки всех фиксируемых метрик
func ReportMetrics(client *http.Client, store *storage.MemStorage, serverAddr string) {
	gauges, counters := store.SnapShot()
	for name, value := range gauges {
		if err := sendMetric(
			client,
			serverAddr,
			"gauge",
			name,
			strconv.FormatFloat(value.Value, 'f', -1, 64)); err != nil {
			log.Printf("Failed to send gauge: %s", err)
		}
	}

	for name, value := range counters {
		if err := sendMetric(
			client,
			serverAddr,
			"counter",
			name, strconv.FormatInt(value.Value, 10)); err != nil {
			log.Printf("Failed to send counter: %s", err)
		}
	}
}
