package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type stubObserver struct {
	called bool
	event  Event
	err    error
}

func (s *stubObserver) Notify(ctx context.Context, event Event) error {
	s.called = true
	s.event = event
	return s.err
}

func TestPublisherNotifyAll(t *testing.T) {
	first := &stubObserver{}
	second := &stubObserver{err: errors.New("boom")}
	publisher := NewPublisher(first, second)
	event := Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}

	err := publisher.Notify(context.Background(), event)
	if err == nil {
		t.Fatalf("expected joined error")
	}
	if !first.called || !second.called {
		t.Fatalf("expected all observers to be called")
	}
	if !reflect.DeepEqual(first.event, event) {
		t.Fatalf("event = %+v, want %+v", first.event, event)
	}
}

func TestFileObserverNotify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	observer := NewFileObserver(path)
	event := Event{TS: 10, Metrics: []string{"Alloc", "PollCount"}, IPAddress: "192.168.0.42"}

	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("second Notify() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}

	var got Event
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, event) {
		t.Fatalf("event = %+v, want %+v", got, event)
	}
}

func TestHTTPObserverNotify(t *testing.T) {
	var (
		gotMethod string
		gotType   string
		gotEvent  Event
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotMethod = request.Method
		gotType = request.Header.Get("Content-Type")
		defer request.Body.Close()
		if err := json.NewDecoder(request.Body).Decode(&gotEvent); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer, err := NewHTTPObserver(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewHTTPObserver() error = %v", err)
	}

	want := Event{TS: 20, Metrics: []string{"HeapInuse"}, IPAddress: "203.0.113.9"}
	if err := observer.Notify(context.Background(), want); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotType != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", gotType, "application/json")
	}
	if !reflect.DeepEqual(gotEvent, want) {
		t.Fatalf("event = %+v, want %+v", gotEvent, want)
	}
}
