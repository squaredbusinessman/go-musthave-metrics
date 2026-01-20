package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

type mockMetricsService struct {
	updateCalled  bool
	updatedMetric models.Metric
	updateErr     error

	getMetricCalled bool
	getMetricArg    models.Metric
	metricValue     string
	metricErr       error

	updateJSONCalled  bool
	updatedMetricJSON models.Metrics
	updateJSONErr     error

	updateBatchCalled   bool
	updatedBatchMetrics []models.Metrics
	updateBatchErr      error

	getMetricJSONCalled bool
	getMetricJSONArg    models.Metrics
	metricJSONResp      *models.Metrics
	metricJSONErr       error

	getAllCalled bool
	gauges       map[string]models.Gauge
	counters     map[string]models.Counter
	allErr       error
}

func newMockMetricsService() *mockMetricsService {
	return &mockMetricsService{
		gauges:   make(map[string]models.Gauge),
		counters: make(map[string]models.Counter),
	}
}

func (m *mockMetricsService) UpdateMetric(ctx context.Context, metric models.Metric) error {
	m.updateCalled = true
	m.updatedMetric = metric
	return m.updateErr
}

func (m *mockMetricsService) UpdateMetricJSON(ctx context.Context, metric models.Metrics) error {
	m.updateJSONCalled = true
	m.updatedMetricJSON = metric
	return m.updateJSONErr
}

func (m *mockMetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	m.updateBatchCalled = true
	m.updatedBatchMetrics = metrics
	return m.updateBatchErr
}

func (m *mockMetricsService) GetMetric(ctx context.Context, metric models.Metric) (string, error) {
	m.getMetricCalled = true
	m.getMetricArg = metric
	if m.metricErr != nil {
		return "", m.metricErr
	}
	return m.metricValue, nil
}

func (m *mockMetricsService) GetMetricJSON(ctx context.Context, metric models.Metrics) (*models.Metrics, error) {
	m.getMetricJSONCalled = true
	m.getMetricJSONArg = metric
	if m.metricJSONErr != nil {
		return nil, m.metricJSONErr
	}
	return m.metricJSONResp, nil
}

func (m *mockMetricsService) GetAllMetrics(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error) {
	m.getAllCalled = true
	if m.gauges == nil {
		m.gauges = map[string]models.Gauge{}
	}
	if m.counters == nil {
		m.counters = map[string]models.Counter{}
	}
	if m.allErr != nil {
		return nil, nil, m.allErr
	}
	return m.gauges, m.counters, nil
}

func TestAcceptMetricsToStorage(t *testing.T) {
	type args struct {
		method string
		target string
		body   string
	}

	tests := []struct {
		name         string
		args         args
		wantCode     int
		expectUpdate bool
		wantMetric   *models.Metric
		setup        func(*mockMetricsService)
	}{
		{
			name: "ok gauge",
			args: args{
				method: http.MethodPost,
				target: "/update/gauge/temperature/42.5",
			},
			wantCode:     http.StatusOK,
			expectUpdate: true,
			wantMetric:   &models.Metric{Type: "gauge", Name: "temperature", Value: "42.5"},
		},
		{
			name: "ok counter",
			args: args{
				method: http.MethodPost,
				target: "/update/counter/requests/10",
			},
			wantCode:     http.StatusOK,
			expectUpdate: true,
			wantMetric:   &models.Metric{Type: "counter", Name: "requests", Value: "10"},
		},
		{
			name: "unsupported type",
			args: args{
				method: http.MethodPost,
				target: "/update/unknown/name/1",
			},
			wantCode:     http.StatusBadRequest,
			expectUpdate: true,
			wantMetric:   &models.Metric{Type: "unknown", Name: "name", Value: "1"},
			setup: func(m *mockMetricsService) {
				m.updateErr = apperr.ErrUnknownMetricType
			},
		},
		{
			name: "bad value gauge",
			args: args{
				method: http.MethodPost,
				target: "/update/gauge/temperature/not-a-number",
			},
			wantCode:     http.StatusBadRequest,
			expectUpdate: true,
			wantMetric:   &models.Metric{Type: "gauge", Name: "temperature", Value: "not-a-number"},
			setup: func(m *mockMetricsService) {
				m.updateErr = apperr.ErrBadMetricValue
			},
		},
		{
			name: "bad value counter",
			args: args{
				method: http.MethodPost,
				target: "/update/counter/requests/not-a-number",
			},
			wantCode:     http.StatusBadRequest,
			expectUpdate: true,
			wantMetric:   &models.Metric{Type: "counter", Name: "requests", Value: "not-a-number"},
			setup: func(m *mockMetricsService) {
				m.updateErr = apperr.ErrBadMetricValue
			},
		},
		{
			name: "wrong method",
			args: args{
				method: http.MethodGet,
				target: "/update/gauge/temperature/1",
			},
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name: "invalid path segments",
			args: args{
				method: http.MethodPost,
				target: "/update/gauge/temperature",
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockMetricsService()
			if tt.setup != nil {
				tt.setup(svc)
			}

			r := chi.NewRouter()
			h := New(svc, nil)
			r.Post("/update/{type}/{name}/{value}", h.AcceptMetricsToStorage)

			req := httptest.NewRequest(tt.args.method, tt.args.target, strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d, body = %q", w.Code, tt.wantCode, w.Body.String())
			}

			if svc.updateCalled != tt.expectUpdate {
				t.Fatalf("updateCalled = %v, want %v", svc.updateCalled, tt.expectUpdate)
			}

			if tt.wantMetric != nil && svc.updateCalled {
				if svc.updatedMetric != *tt.wantMetric {
					t.Fatalf("metric = %+v, want %+v", svc.updatedMetric, *tt.wantMetric)
				}
			}
		})
	}
}

func TestGetMetric(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		setup        func(*mockMetricsService)
		wantCode     int
		wantBody     string
		expectCalled bool
	}{
		{
			name:   "gauge ok",
			method: http.MethodGet,
			path:   "/value/gauge/temp",
			setup: func(m *mockMetricsService) {
				m.metricValue = "10.5"
			},
			wantCode:     http.StatusOK,
			wantBody:     "10.5",
			expectCalled: true,
		},
		{
			name:   "counter ok",
			method: http.MethodGet,
			path:   "/value/counter/poll",
			setup: func(m *mockMetricsService) {
				m.metricValue = "7"
			},
			wantCode:     http.StatusOK,
			wantBody:     "7",
			expectCalled: true,
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   "/value/gauge/miss",
			setup: func(m *mockMetricsService) {
				m.metricErr = apperr.ErrMetricNotFound
			},
			wantCode:     http.StatusNotFound,
			expectCalled: true,
		},
		{
			name:     "bad method",
			method:   http.MethodPost,
			path:     "/value/gauge/temp",
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockMetricsService()
			if tt.setup != nil {
				tt.setup(svc)
			}

			r := chi.NewRouter()
			h := New(svc, nil)
			r.Get("/value/{type}/{name}", h.GetMetric)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if tt.wantBody != "" {
				body := strings.TrimSpace(w.Body.String())
				if body != tt.wantBody {
					t.Fatalf("body = %q, want %q", body, tt.wantBody)
				}
			}

			if svc.getMetricCalled != tt.expectCalled {
				t.Fatalf("getMetricCalled = %v, want %v", svc.getMetricCalled, tt.expectCalled)
			}
		})
	}
}

func TestGetAllMetrics(t *testing.T) {
	svc := newMockMetricsService()
	svc.gauges = map[string]models.Gauge{"Alloc": {Value: 12.3}}
	svc.counters = map[string]models.Counter{"PollCount": {Value: 4}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h := New(svc, nil)
	http.HandlerFunc(h.GetAllMetrics).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	for _, substr := range []string{"Alloc", "12.3", "PollCount", "4"} {
		if !strings.Contains(body, substr) {
			t.Fatalf("response body %q does not contain %q", body, substr)
		}
	}

	if !svc.getAllCalled {
		t.Fatalf("expected GetAllMetrics to be called")
	}
}
