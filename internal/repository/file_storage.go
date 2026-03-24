package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

// ErrFileStoragePathEmpty - путь к файлу хранилища не задан.
var ErrFileStoragePathEmpty = errors.New("file storage path is empty")

// FileStorage - файловая персистентность для метрик.
type FileStorage struct {
	path  string
	store Storage
}

type storageMetricRecord struct {
	ID    string  `json:"id"`
	MType string  `json:"type"`
	Delta int64   `json:"delta,omitempty"`
	Value float64 `json:"value,omitempty"`
}

// NewFileStorage - создает файловое хранилище поверх основного Storage.
func NewFileStorage(path string, store Storage) *FileStorage {
	return &FileStorage{
		path:  path,
		store: store,
	}
}

// Save - сохраняет текущие метрики в файл.
func (fs *FileStorage) Save() error {
	if fs.path == "" {
		return ErrFileStoragePathEmpty
	}

	gauges, counters, err := fs.store.Snapshot(context.Background())
	if err != nil {
		return err
	}

	dir := filepath.Dir(fs.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	} else {
		dir = "."
	}

	tmp, err := os.CreateTemp(dir, "metrics-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := writeMetricsJSON(tmp, gauges, counters); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), fs.path); err != nil {
		return err
	}

	_ = os.Chmod(fs.path, 0o644)

	return nil
}

// Restore - загружает метрики из файла в основное хранилище.
func (fs *FileStorage) Restore() error {
	ctx := context.Background()

	if fs.path == "" {
		return ErrFileStoragePathEmpty
	}

	file, err := os.Open(fs.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		return nil
	}

	decoder := json.NewDecoder(bufio.NewReader(file))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '[' {
		return errors.New("metrics payload must be a JSON array")
	}

	for decoder.More() {
		var metric storageMetricRecord
		if err := decoder.Decode(&metric); err != nil {
			return err
		}

		switch metric.MType {
		case models.MetricTypeGauge:
			if metric.ID == "" {
				continue
			}
			_ = fs.store.SetGauge(ctx, metric.ID, models.Gauge{Value: metric.Value})
		case models.MetricTypeCounter:
			if metric.ID == "" {
				continue
			}
			_ = fs.store.AddCounter(ctx, metric.ID, metric.Delta)
		}
	}

	token, err = decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok = token.(json.Delim)
	if !ok || delimiter != ']' {
		return errors.New("metrics payload must end with a JSON array")
	}

	return nil
}

func writeMetricsJSON(w io.Writer, gauges map[string]models.Gauge, counters map[string]models.Counter) error {
	bw := bufio.NewWriterSize(w, 64*1024)
	if err := bw.WriteByte('['); err != nil {
		return err
	}

	first := true
	record := make([]byte, 0, 128)

	for id, gauge := range gauges {
		var err error
		first, record, err = writeMetricRecord(bw, first, record[:0], id, models.MetricTypeGauge, gauge.Value, 0)
		if err != nil {
			return err
		}
	}

	for id, counter := range counters {
		var err error
		first, record, err = writeMetricRecord(bw, first, record[:0], id, models.MetricTypeCounter, 0, counter.Value)
		if err != nil {
			return err
		}
	}

	if err := bw.WriteByte(']'); err != nil {
		return err
	}
	return bw.Flush()
}

func writeMetricRecord(
	bw *bufio.Writer,
	first bool,
	buf []byte,
	id string,
	metricType string,
	value float64,
	delta int64,
) (bool, []byte, error) {
	if !first {
		if err := bw.WriteByte(','); err != nil {
			return first, buf, err
		}
	}

	buf = append(buf, `{"id":`...)
	buf = strconv.AppendQuote(buf, id)
	buf = append(buf, `,"type":`...)
	buf = strconv.AppendQuote(buf, metricType)

	if metricType == models.MetricTypeGauge {
		buf = append(buf, `,"value":`...)
		buf = strconv.AppendFloat(buf, value, 'f', -1, 64)
	} else {
		buf = append(buf, `,"delta":`...)
		buf = strconv.AppendInt(buf, delta, 10)
	}

	buf = append(buf, '}')
	if _, err := bw.Write(buf); err != nil {
		return first, buf, err
	}

	return false, buf, nil
}
