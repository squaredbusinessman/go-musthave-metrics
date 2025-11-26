package agent

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// SendMetric Функция отправки ОДНОЙ метрики
func SendMetric(client *http.Client, serverAddr string, m models.Metric) error {
	u := url.URL{
		Scheme: "http",
		Host:   serverAddr,
		Path:   path.Join("update", m.Type, m.Name, m.Value),
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
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

	fmt.Printf("Successfully sent metric: %s\n, with value: %s\n\n", m.Name, m.Value)
	return nil
}

// ReportMetrics функция отправки всех фиксируемых метрик
func ReportMetrics(client *http.Client, store *storage.MemStorage, serverAddr string) {
	gauges, counters := store.Snapshot()
	for name, value := range gauges {
		if err := SendMetric(
			client,
			serverAddr,
			models.Metric{
				Type:  "gauge",
				Name:  name,
				Value: strconv.FormatFloat(value.Value, 'f', -1, 64),
			}); err != nil {
			log.Printf("Failed to send gauge: %s", err)
		}
	}

	for name, value := range counters {
		if err := SendMetric(
			client,
			serverAddr,
			models.Metric{
				Type:  "counter",
				Name:  name,
				Value: strconv.FormatInt(value.Value, 10),
			}); err != nil {
			log.Printf("Failed to send counter: %s", err)
		}
	}
}
