package repository

import (
	"reflect"
	"sync"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

func TestNewMemStorage(t *testing.T) {
	storage := NewMemStorage()
	if storage == nil {
		t.Fatal("NewMemStorage returned nil")
	}
	if storage.gauges == nil || storage.counters == nil {
		t.Fatal("maps must be initialised in constructor")
	}
}

func TestMemStorageSetGaugeAndAddCounter(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGauge("Alloc", models.Gauge{Value: 10})
	if got := storage.gauges["Alloc"].Value; got != 10 {
		t.Fatalf("SetGauge stored %v, want 10", got)
	}

	storage.AddCounter("PollCount", 1)
	storage.AddCounter("PollCount", 2)
	if got := storage.counters["PollCount"].Value; got != 3 {
		t.Fatalf("AddCounter aggregated %v, want 3", got)
	}
}

func TestMemStorageSnapshotCopiesMaps(t *testing.T) {
	storage := NewMemStorage()
	storage.SetGauge("RandomValue", models.Gauge{Value: 1})
	storage.AddCounter("PollCount", 1)

	gauges, counters := storage.Snapshot()

	storage.SetGauge("RandomValue", models.Gauge{Value: 2})
	storage.AddCounter("PollCount", 10)

	if gauges["RandomValue"].Value != 1 {
		t.Fatalf("snapshot gauge changed, got %v want 1", gauges["RandomValue"].Value)
	}

	if counters["PollCount"].Value != 1 {
		t.Fatalf("snapshot counter changed, got %v want 1", counters["PollCount"].Value)
	}
}

func TestMemStorage_GetGauge(t *testing.T) {
	type fields struct {
		mutex    sync.RWMutex
		Gauge    models.Gauge
		Counter  models.Counter
		gauges   map[string]models.Gauge
		counters map[string]models.Counter
	}
	type args struct {
		name string
	}
	var tests []struct {
		name   string
		fields fields
		args   args
		want   models.Gauge
		want1  bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				mutex:    tt.fields.mutex,
				Gauge:    tt.fields.Gauge,
				Counter:  tt.fields.Counter,
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got, got1 := ms.GetGauge(tt.args.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGauge() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetGauge() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMemStorage_GetCounter(t *testing.T) {
	type fields struct {
		mutex    sync.RWMutex
		Gauge    models.Gauge
		Counter  models.Counter
		gauges   map[string]models.Gauge
		counters map[string]models.Counter
	}
	type args struct {
		name string
	}
	var tests []struct {
		name   string
		fields fields
		args   args
		want   int64
		want1  bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				mutex:    tt.fields.mutex,
				Gauge:    tt.fields.Gauge,
				Counter:  tt.fields.Counter,
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got, got1 := ms.GetCounter(tt.args.name)
			if got != tt.want {
				t.Errorf("GetCounter() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetCounter() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
