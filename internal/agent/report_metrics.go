package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

const (
	ReportFormatPlain = "plain"
	ReportFormatJSON  = "json"
)

func normalizeReportFormat(format string) string {
	switch strings.ToLower(format) {
	case ReportFormatJSON:
		return ReportFormatJSON
	default:
		return ReportFormatPlain
	}
}

// SendMetric отправляет одну метрику по пути /update/{type}/{name}/{value}.
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

// Отправка метрики в формате json
func sendMetricJSON(client *http.Client, serverAddr string, m models.Metric) error {
	u := url.URL{
		Scheme: "http",
		Host:   serverAddr,
		Path:   "/update",
	}

	metricJSON := models.Metrics{
		ID:    m.Name,
		MType: m.Type,
	}

	switch m.Type {
	case models.MetricTypeGauge:
		val, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return fmt.Errorf("bad gauge value %q: %w", m.Value, err)
		}
		metricJSON.Value = &val
	case models.MetricTypeCounter:
		delta, err := strconv.ParseInt(m.Value, 10, 64)
		if err != nil {
			return fmt.Errorf("bad counter value %q: %w", m.Value, err)
		}
		metricJSON.Delta = &delta
	default:
		return fmt.Errorf("unknown metric type %q", m.Type)
	}

	body, err := json.Marshal(metricJSON)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

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
func ReportMetrics(client *http.Client, store *storage.MemStorage, serverAddr string, reportFormat string) {
	format := normalizeReportFormat(reportFormat)
	sendFunc := SendMetric
	if format == ReportFormatJSON {
		sendFunc = sendMetricJSON
	}

	gauges, counters := store.Snapshot()
	for name, value := range gauges {
		if err := sendFunc(
			client,
			serverAddr,
			models.Metric{
				Type:  models.MetricTypeGauge,
				Name:  name,
				Value: strconv.FormatFloat(value.Value, 'f', -1, 64),
			}); err != nil {
			log.Printf("Failed to send gauge: %s", err)
		}
	}

	for name, value := range counters {
		if err := sendFunc(
			client,
			serverAddr,
			models.Metric{
				Type:  models.MetricTypeCounter,
				Name:  name,
				Value: strconv.FormatInt(value.Value, 10),
			}); err != nil {
			log.Printf("Failed to send counter: %s", err)
		}
	}
}
