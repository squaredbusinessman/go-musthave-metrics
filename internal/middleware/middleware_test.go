package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipMiddlewareCompressesResponse(t *testing.T) {
	const payload = "hello gzip"

	wrapped := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(payload)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))

	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}

	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("new gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(body) != payload {
		t.Fatalf("response body = %q, want %q", string(body), payload)
	}
}

func TestGzipMiddlewareDecompressesRequest(t *testing.T) {
	const payload = "compressed body"

	var received string
	wrapped := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		received = string(data)
		w.WriteHeader(http.StatusOK)
	}))

	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(payload)); err != nil {
		t.Fatalf("write gzip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if received != payload {
		t.Fatalf("handler received %q, want %q", received, payload)
	}
}
