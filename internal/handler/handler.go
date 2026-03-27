package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/audit"
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

// Handler - набор HTTP-хендлеров сервиса метрик.
type Handler struct {
	ms service.MetricsService
	db DBPinger
	// Аудит зависим от HTTP-запроса, потому что только здесь есть IP клиента.
	auditor audit.Notifier
}

// New - создает Handler с зависимостями сервиса, БД и аудита.
func New(ms service.MetricsService, db DBPinger, auditor audit.Notifier) *Handler {
	return &Handler{
		ms:      ms,
		db:      db,
		auditor: auditor,
	}
}

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

// AcceptMetricsToStorage - принимает метрику из URL и сохраняет ее в хранилище.
func (h *Handler) AcceptMetricsToStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if ct := r.Header.Get("Content-Type"); ct != "" && ct != contentTypeTextPlain {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	m := models.Metric{
		Type:  models.MetricType(chi.URLParam(r, urlParamType)),
		Name:  chi.URLParam(r, urlParamName),
		Value: chi.URLParam(r, urlParamValue),
	}

	if m.Type == "" || m.Name == "" || m.Value == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 4 && parts[0] == updatePathPrefix {
			m.Type, m.Name, m.Value = models.MetricType(parts[1]), parts[2], parts[3]
		}
	}

	if m.Type == "" || m.Name == "" || m.Value == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := h.ms.UpdateMetric(r.Context(), m)
	if err != nil {
		apperr.WriteServiceError(w, err)
		return
	}
	h.publishAudit(r, []string{m.Name})
	w.WriteHeader(http.StatusOK)
}

// UpdateMetricJSON - обновляет одну метрику из JSON-запроса.
func (h *Handler) UpdateMetricJSON(writer http.ResponseWriter, request *http.Request) {
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

	err := h.ms.UpdateMetricJSON(request.Context(), req)
	if err != nil {
		apperr.WriteServiceError(writer, err)
		return
	}

	storedMetric, err := h.ms.GetMetricJSON(request.Context(), req)
	if err != nil {
		logger.Log.Error("failed to fetch metric after update", zap.Error(err))
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.publishAudit(request, []string{req.ID})
	writer.Header().Set("Content-Type", contentAppJSON)
	writer.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(writer).Encode(storedMetric); err != nil {
		logger.Log.Error("(update) encode response", zap.Error(err))
	}
}

// UpdateMetricsBatch - обновляет несколько метрик одним JSON-запросом.
func (h *Handler) UpdateMetricsBatch(writer http.ResponseWriter, request *http.Request) {
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
			http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if err := h.ms.UpdateMetricsBatch(request.Context(), req); err != nil {
		apperr.WriteServiceError(writer, err)
		return
	}

	h.publishAudit(request, metricNames(req))
	writer.WriteHeader(http.StatusOK)
}

// GetMetric - возвращает значение метрики в текстовом виде.
func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	m := models.Metric{
		Type: models.MetricType(chi.URLParam(r, urlParamType)),
		Name: chi.URLParam(r, urlParamName),
	}

	if m.Type == "" || m.Name == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	value, err := h.ms.GetMetric(r.Context(), m)
	if err != nil {
		apperr.WriteServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", contentTypeTextPlain)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, value)
}

// GetMetricJSON - возвращает значение метрики в JSON-виде.
func (h *Handler) GetMetricJSON(writer http.ResponseWriter, request *http.Request) {
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

	value, err := h.ms.GetMetricJSON(request.Context(), req)
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

// GetAllMetrics - отдает HTML-страницу со всеми доступными метриками.
func (h *Handler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	gauges, counters, err := h.ms.GetAllMetrics(r.Context())
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

func (h *Handler) publishAudit(request *http.Request, metrics []string) {
	if h.auditor == nil || len(metrics) == 0 {
		return
	}

	event := audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   append([]string(nil), metrics...),
		IPAddress: requestIP(request),
	}

	// Основной запрос уже успешно обработан.
	// Аудит отправляем отдельно и не роняем из-за него ответ клиенту.
	ctx := context.WithoutCancel(request.Context())
	if err := h.auditor.Notify(ctx, event); err != nil {
		logger.Log.Error("audit notify failure",
			zap.Error(err),
			zap.Strings("metrics", event.Metrics),
			zap.String("ip_address", event.IPAddress),
		)
	}
}

func metricNames(metrics []models.Metrics) []string {
	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		names = append(names, metric.ID)
	}
	return names
}

func requestIP(request *http.Request) string {
	if request == nil {
		return ""
	}

	// Если сервер стоит за прокси, сначала берём адрес из заголовков.
	if ip := strings.TrimSpace(request.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}

	if forwarded := strings.TrimSpace(request.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Иначе остаётся адрес TCP-соединения.
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}

	return request.RemoteAddr
}
