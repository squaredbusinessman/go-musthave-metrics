package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
)

const (
	urlParamType  = "type"
	urlParamName  = "name"
	urlParamValue = "value"

	updatePathPrefix = "update"

	contentTypeTextPlain = "text/plain"
	contentTypeHTML      = "text/html; charset=utf-8"
)

// AcceptMetricsToStorage получаем метрики от агента и фиксируем в хранилище
func AcceptMetricsToStorage(ms service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "" && ct != contentTypeTextPlain {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		m := models.Metric{
			Type:  chi.URLParam(r, urlParamType),
			Name:  chi.URLParam(r, urlParamName),
			Value: chi.URLParam(r, urlParamValue),
		}

		if m.Type == "" || m.Name == "" || m.Value == "" {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) == 4 && parts[0] == updatePathPrefix {
				m.Type, m.Name, m.Value = parts[1], parts[2], parts[3]
			}
		}

		if m.Type == "" || m.Name == "" || m.Value == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		err := ms.UpdateMetric(r.Context(), m)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrBadMetricValue):
				http.Error(w, "bad metric value", http.StatusBadRequest)
			case errors.Is(err, service.ErrUnknownMetricType):
				http.Error(w, "unknown metrics type", http.StatusBadRequest)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func GetMetric(ms service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		m := models.Metric{
			Type: chi.URLParam(r, urlParamType),
			Name: chi.URLParam(r, urlParamName),
		}

		if m.Type == "" || m.Name == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		value, err := ms.GetMetric(r.Context(), m)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrMetricNotFound):
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			case errors.Is(err, service.ErrUnknownMetricType):
				http.Error(w, "unknown metrics type", http.StatusBadRequest)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", contentTypeTextPlain)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, value)
	}
}

func GetAllMetrics(ms service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		gauges, counters, err := ms.GetAllMetrics(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

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
