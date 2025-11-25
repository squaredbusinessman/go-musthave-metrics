package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
)

// --- мок хранилища ---
type mockinterfaceStorage interface {
	SetGauge(name string, value models.Gauge)
	AddCounter(name string, value int64)

	GetGauge(name string) (models.Gauge, bool)
	GetCounter(name string) (int64, bool)
	Snapshot() (map[string]models.Gauge, map[string]models.Counter)
}
type mockStorage struct {
	setGaugeCalled   bool
	addCounterCalled bool

	gaugeName  string
	gaugeValue models.Gauge

	counterName  string
	counterValue int64
}

func (m *mockStorage) GetGauge(name string) (models.Gauge, bool) {
	//TODO implement me
	panic("implement me")
}

func (m *mockStorage) GetCounter(name string) (int64, bool) {
	//TODO implement me
	panic("implement me")
}

func (m *mockStorage) SetGauge(name string, value models.Gauge) {
	m.setGaugeCalled = true
	m.gaugeName = name
	m.gaugeValue = value
}

func (m *mockStorage) AddCounter(name string, value int64) {
	m.addCounterCalled = true
	m.counterName = name
	m.counterValue = value
}

func (m *mockStorage) Snapshot() (map[string]models.Gauge, map[string]models.Counter) {
	m.snapshotCalled = true
	return nil, nil
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
			m := &mockStorage{}

			//  http.HandleFunc("/update/", AcceptMetricsToStorage(storage))
			handler := AcceptMetricsToStorage(m)

			req := httptest.NewRequest(tt.args.method, tt.args.target, strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

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
