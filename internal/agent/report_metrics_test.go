package agent

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// конвертация httptest.Server.URL в host:port для SendMetric
func serverAddr(ts *httptest.Server) string {
	parsed, err := url.Parse(ts.URL)
	if err != nil {
		return ts.URL
	}
	return parsed.Host
}

func TestSendMetricSuccess(t *testing.T) {
	var receivedPath, receivedContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedContentType = r.Header.Get("Content-Type")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := ts.Client()
	err := SendMetric(client, serverAddr(ts), models.Metric{
		Type:  "gauge",
		Name:  "Alloc",
		Value: "10",
	})

	if err != nil {
		t.Fatalf("SendMetric returned error: %v", err)
	}

	if receivedPath != "/update/gauge/Alloc/10" {
		t.Fatalf("path = %s, want /update/gauge/Alloc/10", receivedPath)
	}

	if receivedContentType != "text/plain" {
		t.Fatalf("content type = %s, want text/plain", receivedContentType)
	}
}

func TestSendMetricBadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := ts.Client()
	err := SendMetric(client, serverAddr(ts), models.Metric{
		Type:  "gauge",
		Name:  "Alloc",
		Value: "10",
	})
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}
}

type errorRoundTripper struct{ err error }

func (rt errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, rt.err
}

func TestSendMetricHTTPError(t *testing.T) {
	client := &http.Client{Transport: errorRoundTripper{err: errors.New("boom")}}

	err := SendMetric(client, "example.com", models.Metric{
		Type:  "gauge",
		Name:  "Alloc",
		Value: "10",
	})
	if err == nil {
		t.Fatalf("expected error from HTTP client")
	}
}

func TestReportMetricsSendsAllValues(t *testing.T) {
	store := storage.NewMemStorage()
	store.SetGauge("Alloc", models.Gauge{Value: 1})
	store.AddCounter("PollCount", 5)

	var mu sync.Mutex
	requests := make(map[string]int)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.URL.Path]++
		header := r.Header.Get("Content-Type")
		mu.Unlock()

		if header != "text/plain" {
			t.Errorf("Content-Type = %s, want text/plain", header)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ReportMetrics(ts.Client(), store, serverAddr(ts), ReportFormatPlain)

	mu.Lock()
	defer mu.Unlock()

	expected := map[string]int{
		"/update/gauge/Alloc/1":       1,
		"/update/counter/PollCount/5": 1,
	}

	if len(requests) != len(expected) {
		t.Fatalf("got %d requests, want %d", len(requests), len(expected))
	}

	for path, wantCount := range expected {
		if got := requests[path]; got != wantCount {
			t.Fatalf("path %s count = %d, want %d", path, got, wantCount)
		}
	}
}

func TestReportMetricsContinuesAfterError(t *testing.T) {
	store := storage.NewMemStorage()
	store.SetGauge("Alloc", models.Gauge{Value: 1})
	store.AddCounter("PollCount", 5)

	var callCount int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/update/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ReportMetrics(ts.Client(), store, serverAddr(ts), ReportFormatPlain)

	if callCount != 2 {
		t.Fatalf("ReportMetrics should attempt both metrics even after error, got %d calls", callCount)
	}
}

func TestReportMetricsJSONFormat(t *testing.T) {
	store := storage.NewMemStorage()
	store.SetGauge("Alloc", models.Gauge{Value: 2.5})
	store.AddCounter("PollCount", 3)

	var mu sync.Mutex
	payloads := make(map[string]models.Metrics)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %s, want application/json", ct)
		}

		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		mu.Lock()
		payloads[m.ID] = m
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ReportMetrics(ts.Client(), store, serverAddr(ts), ReportFormatJSON)

	mu.Lock()
	defer mu.Unlock()

	if len(payloads) != 2 {
		t.Fatalf("want two payloads, got %d", len(payloads))
	}

	gauge, ok := payloads["Alloc"]
	if !ok || gauge.Value == nil || *gauge.Value != 2.5 {
		t.Fatalf("gauge payload mismatch: %+v", gauge)
	}

	counter, ok := payloads["PollCount"]
	if !ok || counter.Delta == nil || *counter.Delta != 3 {
		t.Fatalf("counter payload mismatch: %+v", counter)
	}
}
