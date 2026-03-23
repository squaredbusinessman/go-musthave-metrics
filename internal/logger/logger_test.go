package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingWriterWriteHeaderAndWrite(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &LoggingWriter{ResponseWriter: recorder}

	writer.WriteHeader(http.StatusCreated)
	n, err := writer.Write([]byte("payload"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if n != len("payload") {
		t.Fatalf("Write() n = %d, want %d", n, len("payload"))
	}
	if writer.Status != http.StatusCreated {
		t.Fatalf("Status = %d, want %d", writer.Status, http.StatusCreated)
	}
	if writer.Bytes != len("payload") {
		t.Fatalf("Bytes = %d, want %d", writer.Bytes, len("payload"))
	}
}

func TestLoggingWriterWriteDefaultsStatusOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &LoggingWriter{ResponseWriter: recorder}

	if _, err := writer.Write([]byte("payload")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if writer.Status != http.StatusOK {
		t.Fatalf("Status = %d, want %d", writer.Status, http.StatusOK)
	}
}

func TestInitialize(t *testing.T) {
	if err := Initialize("debug"); err != nil {
		t.Fatalf("Initialize(debug) error = %v", err)
	}
	if Log == nil {
		t.Fatal("Log must be initialised")
	}
	if err := Initialize("invalid-level"); err == nil {
		t.Fatal("Initialize(invalid-level) must return error")
	}
}
