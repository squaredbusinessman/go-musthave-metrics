package handler

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

const (
	urlParamType  = "type"
	urlParamName  = "name"
	urlParamValue = "value"

	updatePathPrefix = "update"

	contentTypeTextPlain = "text/plain"
	contentAppJSON       = "application/json"
	contentTypeHTML      = "text/html; charset=utf-8"
)

func isJSONContentType(value string) bool {
	if value == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return mediaType == contentAppJSON
}

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
			apperr.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricJSON(ms service.MetricsService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := request.Header.Get("Content-Type"); ct != "" && !isJSONContentType(ct) {
			http.Error(writer, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		logger.Log.Debug("decoding request (update)")
		var req models.Metrics
		defer request.Body.Close()
		decoder := json.NewDecoder(request.Body)
		if err := decoder.Decode(&req); err != nil {
			logger.Log.Error("cannot decode request (update) JSON body", zap.Error(err))
			http.Error(writer, "bad JSON", http.StatusBadRequest)
			return
		}

		if req.ID == "" || req.MType == "" {
			http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		err := ms.UpdateMetricJSON(request.Context(), req)
		if err != nil {
			apperr.WriteServiceError(writer, err)
			return
		}

		storedMetric, err := ms.GetMetricJSON(request.Context(), req)
		if err != nil {
			logger.Log.Error("failed to fetch metric after update", zap.Error(err))
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", contentAppJSON)
		writer.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(writer).Encode(storedMetric); err != nil {
			logger.Log.Error("(update) encode response", zap.Error(err))
		}
	}
}

func UpdateMetricsBatch(ms service.MetricsService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := request.Header.Get("Content-Type"); ct != "" && !isJSONContentType(ct) {
			http.Error(writer, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		logger.Log.Debug("decoding request (updates)")
		var req []models.Metrics
		defer request.Body.Close()
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			logger.Log.Error("cannot decode request (updates) JSON body", zap.Error(err))
			http.Error(writer, "bad JSON", http.StatusBadRequest)
			return
		}

		for _, metric := range req {
			if metric.ID == "" || metric.MType == "" {
				http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
		}

		if err := ms.UpdateMetricsBatch(request.Context(), req); err != nil {
			apperr.WriteServiceError(writer, err)
			return
		}

		writer.WriteHeader(http.StatusOK)
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
			apperr.WriteServiceError(w, err)
			return
		}

		w.Header().Set("Content-Type", contentTypeTextPlain)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, value)
	}
}

func GetMetricJSON(ms service.MetricsService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if ct := request.Header.Get("Content-Type"); ct != "" && !isJSONContentType(ct) {
			http.Error(writer, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		logger.Log.Debug("decoding request (value)")
		var req models.Metrics
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			logger.Log.Error("cannot decode request (value) JSON body", zap.Error(err))
			http.Error(writer, "bad JSON", http.StatusBadRequest)
			return
		}

		defer request.Body.Close()

		if req.MType == "" || req.ID == "" {
			http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		value, err := ms.GetMetricJSON(request.Context(), req)
		if err != nil {
			apperr.WriteServiceError(writer, err)
			return
		}

		writer.Header().Set("Content-Type", contentAppJSON)
		writer.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(writer).Encode(value); err != nil {
			logger.Log.Error("(value) encode response", zap.Error(err))
		}
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
