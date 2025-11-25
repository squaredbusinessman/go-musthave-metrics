package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// AcceptMetricsToStorage получаем метрики от агента и фиксируем в хранилище
func AcceptMetricsToStorage(storage storage.Storage) http.HandlerFunc {
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
			storage.SetGauge(name, models.Gauge{Value: val})
		case `counter`:
			val, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "bad counter value", http.StatusBadRequest)
				return
			}
			storage.AddCounter(name, val)
		default:
			http.Error(w, "unknown metrics type", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func GetMetric(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != `GET` {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, `/`), `/`)
		if len(parts) != 3 || parts[0] != `value` {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		metricType, name := parts[1], parts[2]

		w.Header().Set("Content-Type", "text/plain")

		switch metricType {
		case `gauge`:
			g, ok := storage.GetGauge(name)
			if !ok {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%f", g)
		case `counter`:
			c, ok := storage.GetCounter(name)
			if !ok {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%d", c)
		default:
			http.Error(w, "unknown metrics type", http.StatusBadRequest)
			return
		}
	}
}

func GetAllMetrics(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != `GET` {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		gauges, counters := storage.Snapshot()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "<html><body>")
		fmt.Fprintf(w, "<h1>All metrics</h1>")
		fmt.Fprintf(w, "<table border=\"1\">")
		fmt.Fprintf(w, "<tr><th>Type</th><th>Name</th><th>Value</th></tr>")

		for name, g := range gauges {
			fmt.Fprintf(w, "<tr><td>gauge</td><td>%s</td><td>%v</td></tr>", name, g)
		}
		for name, c := range counters {
			fmt.Fprintf(w, "<tr><td>counter</td><td>%s</td><td>%d</td></tr>", name, c)
		}

		fmt.Fprintf(w, "</table></body></html>")
	}
}
