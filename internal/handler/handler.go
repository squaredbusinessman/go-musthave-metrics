package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

const (
	metricTypeGauge   = "gauge"
	metricTypeCounter = "counter"

	urlParamType  = "type"
	urlParamName  = "name"
	urlParamValue = "value"

	updatePathPrefix = "update"

	contentTypeTextPlain = "text/plain"
	contentTypeHTML      = "text/html; charset=utf-8"
)

// AcceptMetricsToStorage получаем метрики от агента и фиксируем в хранилище
func AcceptMetricsToStorage(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "" && ct != contentTypeTextPlain {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		metricType := chi.URLParam(r, urlParamType)
		metricName := chi.URLParam(r, urlParamName)
		metricValue := chi.URLParam(r, urlParamValue)

		if metricType == "" || metricName == "" || metricValue == "" {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) == 4 && parts[0] == updatePathPrefix {
				metricType, metricName, metricValue = parts[1], parts[2], parts[3]
			}
		}

		if metricType == "" || metricName == "" || metricValue == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		switch metricType {
		case metricTypeGauge:
			val, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "bad gauge value", http.StatusBadRequest)
				return
			}
			storage.SetGauge(metricName, models.Gauge{Value: val})
		case metricTypeCounter:
			val, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "bad counter value", http.StatusBadRequest)
				return
			}
			storage.AddCounter(metricName, val)
		default:
			http.Error(w, "unknown metrics type", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func GetMetric(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		metricType := chi.URLParam(r, urlParamType)
		metricName := chi.URLParam(r, urlParamName)

		if metricType == "" || metricName == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", contentTypeTextPlain)

		switch metricType {
		case metricTypeGauge:
			g, ok := storage.GetGauge(metricName)
			if !ok {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%s", strconv.FormatFloat(g, 'f', -1, 64))
		case metricTypeCounter:
			c, ok := storage.GetCounter(metricName)
			if !ok {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%s", strconv.FormatInt(c, 10))
		default:
			http.Error(w, "unknown metrics type", http.StatusBadRequest)
			return
		}
	}
}

func GetAllMetrics(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		gauges, counters := storage.Snapshot()

		w.Header().Set("Content-Type", contentTypeHTML)
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "<html><body>")
		fmt.Fprintf(w, "<h1>All metrics</h1>")
		fmt.Fprintf(w, "<table border=\"1\">")
		fmt.Fprintf(w, "<tr><th>Type</th><th>Name</th><th>Value</th></tr>")

		for name, g := range gauges {
			fmt.Fprintf(w, "<tr><td>gauge</td><td>%s</td><td>%v</td></tr>", name, g.Value)
		}
		for name, c := range counters {
			fmt.Fprintf(w, "<tr><td>counter</td><td>%s</td><td>%d</td></tr>", name, c.Value)
		}

		fmt.Fprintf(w, "</table></body></html>")
	}
}
