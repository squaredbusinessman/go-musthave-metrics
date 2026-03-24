package logger

import (
	"net/http"

	"go.uber.org/zap"
)

// LoggingWriter - обертка над ResponseWriter для логирования статуса и размера ответа.
type LoggingWriter struct {
	http.ResponseWriter
	Status int
	Bytes  int
}

// WriteHeader - запоминает HTTP-статус и передает его дальше.
func (lw *LoggingWriter) WriteHeader(code int) {
	lw.Status = code
	lw.ResponseWriter.WriteHeader(code)
}

// Write - пишет тело ответа и считает количество байт.
func (lw *LoggingWriter) Write(b []byte) (int, error) {
	n, err := lw.ResponseWriter.Write(b)
	lw.Bytes += n
	if lw.Status == 0 {
		lw.Status = http.StatusOK
	}
	return n, err
}

// Log - общий logger проекта.
var Log *zap.Logger = zap.NewNop()

// Initialize - настраивает общий logger по указанному уровню.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()

	cfg.Level = lvl

	zapLog, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zapLog
	return nil
}
