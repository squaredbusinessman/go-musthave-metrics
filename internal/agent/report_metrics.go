package agent

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"go.uber.org/zap"
)

const (
	ReportFormatPlain = "plain"
	ReportFormatJSON  = "json"
	updatePath        = "/update"
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
func SendMetric(client *resty.Client, m models.Metric) error {

	postPath := path.Join("update", m.Type, m.Name, m.Value)

	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(postPath)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("bad status: %s", resp.Status())
	}

	myLog.Log.
		Info("Metric sent",
			zap.String("metric", m.Name),
			zap.String("value", m.Value),
			zap.String("format", ReportFormatPlain))
	return nil
}

// Отправка метрики в формате json
func sendMetricJSON(client *resty.Client, m models.Metric) error {

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

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metricJSON).
		Post(updatePath)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("bad status: %s", resp.Status())
	}

	myLog.Log.
		Info("Metric sent",
			zap.String("metric", m.Name),
			zap.String("value", m.Value),
			zap.String("format", ReportFormatJSON))
	return nil
}

// ReportMetrics функция отправки всех фиксируемых метрик
func ReportMetrics(client *resty.Client, store *storage.MemStorage, reportFormat string) {
	format := normalizeReportFormat(reportFormat)
	sendFunc := SendMetric
	if format == ReportFormatJSON {
		sendFunc = sendMetricJSON
	}

	gauges, counters, err := store.Snapshot(context.Background())
	if err != nil {
		myLog.Log.Warn("Failed to snapshot metrics", zap.Error(err))
		return
	}
	for name, value := range gauges {
		if err := sendFunc(
			client,
			models.Metric{
				Type:  models.MetricTypeGauge,
				Name:  name,
				Value: strconv.FormatFloat(value.Value, 'f', -1, 64),
			}); err != nil {
			myLog.Log.Warn("Failed to send gauge", zap.Error(err))
		}
	}

	for name, value := range counters {
		if err := sendFunc(
			client,
			models.Metric{
				Type:  models.MetricTypeCounter,
				Name:  name,
				Value: strconv.FormatInt(value.Value, 10),
			}); err != nil {
			myLog.Log.Warn("Failed to send counter", zap.Error(err))
		}
	}
}
