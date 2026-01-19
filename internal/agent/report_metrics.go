package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	updatesPath       = "/updates"
)

var errBatchUnsupported = errors.New("batch updates not supported")

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

func sendMetricJSON(client *resty.Client, metric models.Metrics) error {
	payload, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	body, err := gzipPayload(payload)
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body).
		Post(updatePath)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("bad status: %s", resp.Status())
	}

	myLog.Log.Info("Metric sent", zap.String("metric", metric.ID), zap.String("format", ReportFormatJSON))
	return nil
}

func sendMetricsBatchJSON(client *resty.Client, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	payload, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	body, err := gzipPayload(payload)
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body).
		Post(updatesPath)
	if err != nil {
		return err
	}

	if resp.IsSuccess() {
		myLog.Log.Info("Metrics batch sent", zap.Int("count", len(metrics)), zap.String("format", ReportFormatJSON))
		return nil
	}

	switch resp.StatusCode() {
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return errBatchUnsupported
	default:
		return fmt.Errorf("bad status: %s", resp.Status())
	}
}

func gzipPayload(payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func snapshotToMetrics(gauges map[string]models.Gauge, counters map[string]models.Counter) []models.Metrics {
	metrics := make([]models.Metrics, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		v := value.Value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.MetricTypeGauge,
			Value: &v,
		})
	}
	for name, value := range counters {
		d := value.Value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.MetricTypeCounter,
			Delta: &d,
		})
	}
	return metrics
}

// ReportMetrics функция отправки всех фиксируемых метрик
func ReportMetrics(client *resty.Client, store *storage.MemStorage, reportFormat string) {
	format := normalizeReportFormat(reportFormat)

	gauges, counters, err := store.Snapshot(context.Background())
	if err != nil {
		myLog.Log.Warn("Failed to snapshot metrics", zap.Error(err))
		return
	}

	if format == ReportFormatJSON {
		metrics := snapshotToMetrics(gauges, counters)
		if len(metrics) == 0 {
			return
		}

		if err := sendMetricsBatchJSON(client, metrics); err != nil {
			if errors.Is(err, errBatchUnsupported) {
				for _, metric := range metrics {
					if err := sendMetricJSON(client, metric); err != nil {
						myLog.Log.Warn("Failed to send metric (fallback)", zap.Error(err))
					}
				}
				return
			}

			myLog.Log.Warn("Failed to send metrics batch", zap.Error(err))
		}
		return
	}

	for name, value := range gauges {
		if err := SendMetric(
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
		if err := SendMetric(
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
