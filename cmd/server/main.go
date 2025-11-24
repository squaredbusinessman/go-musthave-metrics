package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Storage interface {
	SetGauge(name string, value Gauge)
	SetCounter(name string, value Counter)
}
type Gauge struct {
	value float64
}

func (g *Gauge) NewGaugeReplace(val float64) {
	g.value = val
}

type Counter struct {
	value int64
}

func (c *Counter) NewValueIncrement(val int64) {
	c.value += val
}

type MemStorage struct {
	Gauge
	Counter
	gauges   map[string]Gauge
	counters map[string]Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]Gauge),
		counters: make(map[string]Counter),
	}
}

func (ms *MemStorage) SetGauge(name string, value Gauge) {
	ms.gauges[name] = value
}

func (ms *MemStorage) SetCounter(name string, value Counter) {
	ms.counters[name] = value
}

func main() {
	metricsStorage := NewMemStorage()

	if err := http.ListenAndServe(`:8080`, AcceptMetricsToStorage(metricsStorage)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Serving metrics at http://localhost:8080/metrics")
}

func AcceptMetricsToStorage(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != `POST` {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != `text/plain` {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		}

		parts := strings.Split(strings.Trim(r.URL.Path, `/`), `/`)
		if len(parts) != 4 || parts[0] != `update` {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		metricType, name, raw := parts[1], parts[2], parts[3]
		switch metricType {
		case `gauge`:
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				http.Error(w, "bad gauge value", http.StatusBadRequest)
				return
			}
			storage.SetGauge(name, Gauge{val})
		case `counter`:
			val, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "bad counter value", http.StatusBadRequest)
				return
			}
			storage.SetCounter(name, Counter{val})
		default:
			http.Error(w, "unknown metrics type", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
