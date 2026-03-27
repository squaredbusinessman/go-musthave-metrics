package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewDBStorage(t *testing.T) {
	storage := NewDBStorage(nil)
	if storage == nil {
		t.Fatal("NewDBStorage() returned nil")
	}
	if storage.pool != nil {
		t.Fatal("NewDBStorage(nil) must keep nil pool")
	}
}

func TestBuildQueries(t *testing.T) {
	tests := []struct {
		name      string
		build     func() (string, []any, error)
		wantParts []string
		wantArgs  []any
	}{
		{
			name: "upsert gauge",
			build: func() (string, []any, error) {
				return buildUpsertGauge("Alloc", 42.5)
			},
			wantParts: []string{
				"INSERT INTO gauges",
				"ON CONFLICT (metric_name) DO UPDATE SET value = EXCLUDED.value",
			},
			wantArgs: []any{"Alloc", 42.5},
		},
		{
			name: "upsert counter",
			build: func() (string, []any, error) {
				return buildUpsertCounter("PollCount", 7)
			},
			wantParts: []string{
				"INSERT INTO counters",
				"DO UPDATE SET value = counters.value + EXCLUDED.value",
			},
			wantArgs: []any{"PollCount", int64(7)},
		},
		{
			name: "get gauge",
			build: func() (string, []any, error) {
				return buildGetGauge("Alloc")
			},
			wantParts: []string{"SELECT value FROM gauges", "WHERE metric_name = $1"},
			wantArgs:  []any{"Alloc"},
		},
		{
			name: "get counter",
			build: func() (string, []any, error) {
				return buildGetCounter("PollCount")
			},
			wantParts: []string{"SELECT value FROM counters", "WHERE metric_name = $1"},
			wantArgs:  []any{"PollCount"},
		},
		{
			name: "snapshot gauges",
			build: func() (string, []any, error) {
				return buildSnapshotGauges()
			},
			wantParts: []string{"SELECT metric_name, value FROM gauges"},
		},
		{
			name: "snapshot counters",
			build: func() (string, []any, error) {
				return buildSnapshotCounters()
			},
			wantParts: []string{"SELECT metric_name, value FROM counters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.build()
			if err != nil {
				t.Fatalf("build() error = %v", err)
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(sql, part) {
					t.Fatalf("sql = %q, want to contain %q", sql, part)
				}
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args len = %d, want %d", len(args), len(tt.wantArgs))
			}
			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Fatalf("arg[%d] = %v, want %v", i, args[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestIsRetryablePGErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{name: "retryable pg error", err: &pgconn.PgError{Code: "08006"}, want: true},
		{name: "non retryable pg error", err: &pgconn.PgError{Code: "23505"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryablePGErr(tt.err); got != tt.want {
				t.Fatalf("isRetryablePGErr() = %v, want %v", got, tt.want)
			}
		})
	}
}
