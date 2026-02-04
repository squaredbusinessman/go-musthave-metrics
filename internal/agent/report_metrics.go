package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/retry"
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

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

// SendMetric отправляет одну метрику по пути /update/{type}/{name}/{value}.
func SendMetric(client *resty.Client, m models.Metric, key string) error {
	postPath := path.Join("update", m.Type, m.Name, m.Value)

	if err := retry.Do(context.Background(), isRetryableNetErr, func() error {
		req := client.R().
			SetHeader("Content-Type", "text/plain")
		if key != "" {
			req.SetHeader("HashSHA256", sha256hex(nil, key))
		}
		resp, err := req.Post(postPath)
		if err != nil {
			return err
		}
		if !resp.IsSuccess() {
			return fmt.Errorf("bad status: %s", resp.Status())
		}
		return nil
	}); err != nil {
		return err
	}

	myLog.Log.
		Info("Metric sent",
			zap.String("metric", m.Name),
			zap.String("value", m.Value),
			zap.String("format", ReportFormatPlain))
	return nil
}

func sendMetricJSON(client *resty.Client, metric models.Metrics, key string) error {
	payload, err := json.Marshal(metric)
	if err != nil {
		return err
	}
	body, err := gzipPayload(payload)
	if err != nil {
		return err
	}

	if err := retry.Do(context.Background(), isRetryableNetErr, func() error {
		req := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body)
		if key != "" {
			req.SetHeader("HashSHA256", sha256hex(payload, key))
		}
		resp, err := req.Post(updatePath)
		if err != nil {
			return err
		}
		if !resp.IsSuccess() {
			return fmt.Errorf("bad status: %s", resp.Status())
		}
		return nil
	}); err != nil {
		return err
	}

	myLog.Log.Info("Metric sent", zap.String("metric", metric.ID), zap.String("format", ReportFormatJSON))
	return nil
}

func sendMetricsBatchJSON(client *resty.Client, metrics []models.Metrics, key string) error {
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

	if err := retry.Do(context.Background(), isRetryableNetErr, func() error {
		req := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body)
		if key != "" {
			req.SetHeader("HashSHA256", sha256hex(payload, key))
		}
		resp, err := req.Post(updatesPath)
		if err != nil {
			return err
		}

		if resp.IsSuccess() {
			return nil
		}

		switch resp.StatusCode() {
		case http.StatusNotFound, http.StatusMethodNotAllowed:
			return errBatchUnsupported
		default:
			return fmt.Errorf("bad status: %s", resp.Status())
		}
	}); err != nil {
		return err
	}

	myLog.Log.Info("Metrics batch sent", zap.Int("count", len(metrics)), zap.String("format", ReportFormatJSON))
	return nil
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
func ReportMetrics(client *resty.Client, store *storage.MemStorage, reportFormat string, key string, jobs chan<- Job) {
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

		jobs <- func() error {
			if err = sendMetricsBatchJSON(client, metrics, key); err != nil {
				if errors.Is(err, errBatchUnsupported) {
					for _, metric := range metrics {
						if err := sendMetricJSON(client, metric, key); err != nil {
							myLog.Log.Warn("Failed to send metric (fallback)", zap.Error(err))
						}
					}
					return nil
				}
				return err
			}
			return nil
		}
		return
	}

	for name, value := range gauges {
		n := name
		v := value.Value
		jobs <- func() error {
			return SendMetric(client, models.Metric{
				Type:  models.MetricTypeGauge,
				Name:  n,
				Value: strconv.FormatFloat(v, 'f', -1, 64),
			}, key)
		}
	}

	for name, value := range counters {
		n := name
		v := value.Value
		jobs <- func() error {
			return SendMetric(client, models.Metric{
				Type:  models.MetricTypeCounter,
				Name:  n,
				Value: strconv.FormatInt(v, 10),
			}, key)
		}
	}
}
