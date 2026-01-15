package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

var ErrFileStoragePathEmpty = errors.New("file storage path is empty")

type FileStorage struct {
	path  string
	store Storage
}

func NewFileStorage(path string, store Storage) *FileStorage {
	return &FileStorage{
		path:  path,
		store: store,
	}
}

func (fs *FileStorage) Save() error {
	if fs.path == "" {
		return ErrFileStoragePathEmpty
	}

	gauges, counters, err := fs.store.Snapshot(context.Background())
	if err != nil {
		return err
	}

	metrics := make([]models.Metrics, 0, len(gauges)+len(counters))
	for id, gauge := range gauges {
		value := gauge.Value
		metrics = append(metrics, models.Metrics{
			ID:    id,
			MType: "gauge",
			Value: &value,
		})
	}
	for id, counter := range counters {
		delta := counter.Value
		metrics = append(metrics, models.Metrics{
			ID:    id,
			MType: "counter",
			Delta: &delta,
		})
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
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

	if _, err := tmp.Write(data); err != nil {
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

func (fs *FileStorage) Restore() error {
	ctx := context.Background()

	if fs.path == "" {
		return ErrFileStoragePathEmpty
	}

	data, err := os.ReadFile(fs.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.MetricTypeGauge:
			if metric.Value == nil {
				continue
			}
			_ = fs.store.SetGauge(ctx, metric.ID, models.Gauge{Value: *metric.Value})
		case models.MetricTypeCounter:
			if metric.Delta == nil {
				continue
			}
			_ = fs.store.AddCounter(ctx, metric.ID, *metric.Delta)
		}
	}

	return nil
}
