package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/cryptoutil"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

func newTestClient(ts *httptest.Server) *resty.Client {
	return resty.New().SetBaseURL(ts.URL)
}

func runJobs(jobs <-chan Job) {
	for job := range jobs {
		if job == nil {
			continue
		}
		_ = job()
	}
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

	client := newTestClient(ts)
	err := SendMetric(client, models.Metric{
		Type:  "gauge",
		Name:  "Alloc",
		Value: "10",
	}, "")

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

	client := newTestClient(ts)
	err := SendMetric(client, models.Metric{
		Type:  "gauge",
		Name:  "Alloc",
		Value: "10",
	}, "")
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}
}

type errorRoundTripper struct{ err error }

func (rt errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, rt.err
}

func TestSendMetricHTTPError(t *testing.T) {
	client := resty.New().
		SetBaseURL("http://example.com").
		SetTransport(errorRoundTripper{err: errors.New("boom")})

	err := SendMetric(client, models.Metric{
		Type:  "gauge",
		Name:  "Alloc",
		Value: "10",
	}, "")
	if err == nil {
		t.Fatalf("expected error from HTTP client")
	}
}

func TestReportMetricsSendsAllValues(t *testing.T) {
	store := storage.NewMemStorage()
	ctx := context.Background()
	if err := store.SetGauge(ctx, "Alloc", models.Gauge{Value: 1}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := store.AddCounter(ctx, "PollCount", 5); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

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

	jobs := make(chan Job, 100)
	ReportMetrics(newTestClient(ts), store, ReportFormatPlain, "", nil, jobs)
	close(jobs)
	runJobs(jobs)

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
	ctx := context.Background()
	if err := store.SetGauge(ctx, "Alloc", models.Gauge{Value: 1}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := store.AddCounter(ctx, "PollCount", 5); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

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

	jobs := make(chan Job, 100)
	ReportMetrics(newTestClient(ts), store, ReportFormatPlain, "", nil, jobs)
	close(jobs)
	runJobs(jobs)

	if callCount != 2 {
		t.Fatalf("ReportMetrics should attempt both metrics even after error, got %d calls", callCount)
	}
}

func TestReportMetricsJSONFormat(t *testing.T) {
	store := storage.NewMemStorage()
	ctx := context.Background()
	if err := store.SetGauge(ctx, "Alloc", models.Gauge{Value: 2.5}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}
	if err := store.AddCounter(ctx, "PollCount", 3); err != nil {
		t.Fatalf("AddCounter() error = %v", err)
	}

	var mu sync.Mutex
	callCount := 0
	payloads := make(map[string]models.Metrics)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()

		if r.URL.Path != "/updates" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %s, want application/json", ct)
		}
		if enc := r.Header.Get("Content-Encoding"); enc != "gzip" {
			t.Fatalf("Content-Encoding = %s, want gzip", enc)
		}

		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer reader.Close()

		var batch []models.Metrics
		if err := json.NewDecoder(reader).Decode(&batch); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		mu.Lock()
		for _, metric := range batch {
			payloads[metric.ID] = metric
		}
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	jobs := make(chan Job, 100)
	ReportMetrics(newTestClient(ts), store, ReportFormatJSON, "", nil, jobs)
	close(jobs)
	runJobs(jobs)

	mu.Lock()
	defer mu.Unlock()

	if callCount != 1 {
		t.Fatalf("want one batch request, got %d", callCount)
	}

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

func TestReportMetricsEncryptsJSONBatch(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	store := storage.NewMemStorage()
	ctx := context.Background()
	if err := store.SetGauge(ctx, "Alloc", models.Gauge{Value: 2.5}); err != nil {
		t.Fatalf("SetGauge() error = %v", err)
	}

	var gotMetric models.Metrics
	var gotPath string
	var gotEncryption string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotEncryption = r.Header.Get("Content-Encryption")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if bytes.Contains(body, []byte("Alloc")) {
			t.Fatal("encrypted body contains plaintext metric name")
		}

		decrypted, err := cryptoutil.Decrypt(privateKey, body)
		if err != nil {
			t.Fatalf("decrypt body: %v", err)
		}

		reader, err := gzip.NewReader(bytes.NewReader(decrypted))
		if err != nil {
			t.Fatalf("create gzip reader: %v", err)
		}
		defer reader.Close()

		var batch []models.Metrics
		if err := json.NewDecoder(reader).Decode(&batch); err != nil {
			t.Fatalf("decode json: %v", err)
		}

		if len(batch) != 1 {
			t.Fatalf("batch len = %d, want 1", len(batch))
		}

		gotMetric = batch[0]
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	jobs := make(chan Job, 100)
	ReportMetrics(newTestClient(ts), store, ReportFormatPlain, "", &privateKey.PublicKey, jobs)
	close(jobs)
	runJobs(jobs)

	if gotPath != "/updates" {
		t.Fatalf("path = %s, want /updates", gotPath)
	}

	if gotEncryption != "rsa-aes-gcm" {
		t.Fatalf("Content-Encryption = %s, want rsa-aes-gcm", gotEncryption)
	}

	if gotMetric.ID != "Alloc" || gotMetric.Value == nil || *gotMetric.Value != 2.5 {
		t.Fatalf("metric mismatch: %+v", gotMetric)
	}
}
