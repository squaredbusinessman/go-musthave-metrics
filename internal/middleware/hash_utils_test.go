package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSHA256Hex(t *testing.T) {
	sum := sha256.Sum256([]byte("payloadsecret"))
	want := hex.EncodeToString(sum[:])

	if got := sha256hex([]byte("payload"), "secret"); got != want {
		t.Fatalf("sha256hex() = %q, want %q", got, want)
	}
}

func TestResponseRecorderFlushTo(t *testing.T) {
	dst := httptest.NewRecorder()
	rec := NewRecorder(dst)
	rec.Header().Add("X-Test", "value")
	rec.WriteHeader(http.StatusAccepted)

	if _, err := rec.Write([]byte("buffered body")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := string(rec.Body()); got != "buffered body" {
		t.Fatalf("Body() = %q, want %q", got, "buffered body")
	}

	rec.FlushTo(dst)

	resp := dst.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	if got := resp.Header.Get("X-Test"); got != "value" {
		t.Fatalf("X-Test = %q, want %q", got, "value")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(body); got != "buffered body" {
		t.Fatalf("flushed body = %q, want %q", got, "buffered body")
	}
}

func TestHashMiddlewareRejectsBadHash(t *testing.T) {
	called := false
	handler := HashMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("payload"))
	req.Header.Set("HashSHA256", "bad-hash")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next handler must not be called for invalid hash")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHashMiddlewareValidatesAndSignsResponse(t *testing.T) {
	var gotRequestBody string

	handler := HashMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		gotRequestBody = string(data)
		if _, err = w.Write([]byte("response")); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}))

	reqBody := "payload"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("HashSHA256", sha256hex([]byte(reqBody), "secret"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if gotRequestBody != reqBody {
		t.Fatalf("handler body = %q, want %q", gotRequestBody, reqBody)
	}
	if got := resp.Header.Get("HashSHA256"); got != sha256hex([]byte("response"), "secret") {
		t.Fatalf("response hash = %q, want %q", got, sha256hex([]byte("response"), "secret"))
	}
}

func TestConveyorAppliesMiddlewaresInOrder(t *testing.T) {
	sequence := ""

	makeMiddleware := func(tag string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sequence += tag
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := Conveyor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence += "handler"
		w.WriteHeader(http.StatusNoContent)
	}), makeMiddleware("first>"), makeMiddleware("second>"))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if sequence != "second>first>handler" {
		t.Fatalf("sequence = %q, want %q", sequence, "second>first>handler")
	}
}
