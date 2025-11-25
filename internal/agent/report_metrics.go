package agent

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// SendMetric Функция отправки ОДНОЙ метрики
func SendMetric(client *http.Client, serverAddr string, metricType string, metricName string, metricValue string) error {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s", serverAddr, metricType, metricName, metricValue)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	fmt.Printf("Successfully sent metric: %s\n, with value: %s\n\n", metricName, metricValue)
	return nil
}

// ReportMetrics функция отправки всех фиксируемых метрик
func ReportMetrics(client *http.Client, store *storage.MemStorage, serverAddr string) {
	gauges, counters := store.SnapShot()
	for name, value := range gauges {
		if err := SendMetric(
			client,
			serverAddr,
			"gauge",
			name,
			strconv.FormatFloat(value.Value, 'f', -1, 64)); err != nil {
			log.Printf("Failed to send gauge: %s", err)
		}
	}

	for name, value := range counters {
		if err := SendMetric(
			client,
			serverAddr,
			"counter",
			name, strconv.FormatInt(value.Value, 10)); err != nil {
			log.Printf("Failed to send counter: %s", err)
		}
	}
}
