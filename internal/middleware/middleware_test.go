package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/cryptoutil"
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

func TestCryptoMiddlewareDecryptsRequest(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	plaintext := []byte(`{"ok":true}`)

	encrypted, err := cryptoutil.Encrypt(&privateKey.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	var gotBody []byte

	handler := CryptoMiddleware(privateKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if enc := r.Header.Get("Content-Encryption"); enc != "" {
			t.Fatalf("Content-Encryption = %q, want empty", enc)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(encrypted))
	req.Header.Set("Content-Encryption", "rsa-aes-gcm")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !bytes.Equal(gotBody, plaintext) {
		t.Fatalf("body = %q, want %q", gotBody, plaintext)
	}
}

// тест на порядок исполнения мидлварин
func TestCryptoMiddlewareBeforeGzipMiddleware(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	payload := []byte(`{"id":"Alloc","type":"gauge","value":42.5}`)

	var gzipped bytes.Buffer
	zw := gzip.NewWriter(&gzipped)
	if _, err := zw.Write(payload); err != nil {
		t.Fatalf("write gzip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	encrypted, err := cryptoutil.Encrypt(&privateKey.PublicKey, gzipped.Bytes())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	var got map[string]any

	handler := CryptoMiddleware(privateKey)(
		GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("decode json: %v", err)
			}

			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(encrypted))
	req.Header.Set("Content-Encryption", "rsa-aes-gcm")
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got["id"] != "Alloc" {
		t.Fatalf("id = %v, want Alloc", got["id"])
	}
}
