package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoRetriesOnRetryable(t *testing.T) {
	orig := backoffs
	backoffs = []time.Duration{0, 0, 0}
	defer func() { backoffs = orig }()

	var calls int
	err := Do(context.Background(), func(err error) bool { return err != nil }, func() error {
		calls++
		if calls < 4 {
			return errors.New("temporary")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 4 {
		t.Fatalf("calls = %d, want 4", calls)
	}
}

func TestDoStopsOnNonRetryable(t *testing.T) {
	orig := backoffs
	backoffs = []time.Duration{0, 0, 0}
	defer func() { backoffs = orig }()

	var calls int
	err := Do(context.Background(), func(error) bool { return false }, func() error {
		calls++
		return errors.New("permanent")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDoStopsOnContextCancel(t *testing.T) {
	orig := backoffs
	backoffs = []time.Duration{0, 0, 0}
	defer func() { backoffs = orig }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int
	err := Do(ctx, func(error) bool { return true }, func() error {
		calls++
		return errors.New("temporary")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0", calls)
	}
}
