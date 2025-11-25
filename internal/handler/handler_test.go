package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

// --- мок хранилища ---
type mockStorage struct {
	setGaugeCalled   bool
	addCounterCalled bool
	snapshotCalled   bool

	gaugeName  string
	gaugeValue models.Gauge

	counterName  string
	counterValue int64

	gauges   map[string]float64
	counters map[string]int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *mockStorage) GetGauge(name string) (float64, bool) {
	val, ok := m.gauges[name]
	return val, ok
}

func (m *mockStorage) GetCounter(name string) (int64, bool) {
	val, ok := m.counters[name]
	return val, ok
}

func (m *mockStorage) SetGauge(name string, value models.Gauge) {
	m.setGaugeCalled = true
	m.gaugeName = name
	m.gaugeValue = value
	m.gauges[name] = value.Value
}

func (m *mockStorage) AddCounter(name string, value int64) {
	m.addCounterCalled = true
	m.counterName = name
	m.counterValue = value
	m.counters[name] += value
}

func (m *mockStorage) Snapshot() (map[string]models.Gauge, map[string]models.Counter) {
	m.snapshotCalled = true
	gauges := make(map[string]models.Gauge, len(m.gauges))
	for k, v := range m.gauges {
		gauges[k] = models.Gauge{Value: v}
	}
	counters := make(map[string]models.Counter, len(m.counters))
	for k, v := range m.counters {
		counters[k] = models.Counter{Value: v}
	}
	return gauges, counters
}

// --- сами тесты ---

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
		wantSetGauge bool
		wantAddCount bool
	}{
		{
			name: "ok gauge",
			args: args{
				method: http.MethodPost,
				target: "/update/gauge/temperature/42.5",
				body:   "",
			},
			wantCode:     http.StatusOK,
			wantSetGauge: true,
		},
		{
			name: "ok counter",
			args: args{
				method: http.MethodPost,
				target: "/update/counter/requests/10",
			},
			wantCode:     http.StatusOK,
			wantAddCount: true,
		},
		{
			name: "unsupported type",
			args: args{
				method: http.MethodPost,
				target: "/update/unknown/name/1",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "bad value gauge",
			args: args{
				method: http.MethodPost,
				target: "/update/gauge/temperature/not-a-number",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "bad value counter",
			args: args{
				method: http.MethodPost,
				target: "/update/counter/requests/not-a-number",
			},
			wantCode: http.StatusBadRequest,
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
				target: "/update/gauge/temperature", // 3 сегмента вместо 4
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMockStorage()

			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", AcceptMetricsToStorage(m))

			req := httptest.NewRequest(tt.args.method, tt.args.target, strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d, body = %q", w.Code, tt.wantCode, w.Body.String())
			}

			if m.setGaugeCalled != tt.wantSetGauge {
				t.Errorf("setGaugeCalled = %v, want %v", m.setGaugeCalled, tt.wantSetGauge)
			}
			if m.addCounterCalled != tt.wantAddCount {
				t.Errorf("addCounterCalled = %v, want %v", m.addCounterCalled, tt.wantAddCount)
			}
		})
	}
}

func TestGetMetric(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		prepare  func(*mockStorage)
		wantCode int
		wantBody string
	}{
		{
			name:   "gauge ok",
			method: http.MethodGet,
			path:   "/value/gauge/temp",
			prepare: func(s *mockStorage) {
				s.gauges["temp"] = 10.5
			},
			wantCode: http.StatusOK,
			wantBody: "10.5",
		},
		{
			name:   "counter ok",
			method: http.MethodGet,
			path:   "/value/counter/poll",
			prepare: func(s *mockStorage) {
				s.counters["poll"] = 7
			},
			wantCode: http.StatusOK,
			wantBody: "7",
		},
		{
			name:     "not found",
			method:   http.MethodGet,
			path:     "/value/gauge/miss",
			wantCode: http.StatusNotFound,
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
			store := newMockStorage()
			if tt.prepare != nil {
				tt.prepare(store)
			}

			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", GetMetric(store))

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
		})
	}
}

func TestGetAllMetrics(t *testing.T) {
	store := newMockStorage()
	store.gauges["Alloc"] = 12.3
	store.counters["PollCount"] = 4

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	GetAllMetrics(store).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	for _, substr := range []string{"Alloc", "12.3", "PollCount", "4"} {
		if !strings.Contains(body, substr) {
			t.Fatalf("response body %q does not contain %q", body, substr)
		}
	}
}
