package agent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPoolWaitDrainsQueue(t *testing.T) {
	pool := NewWorkerPool(2, 4)
	jobs := pool.Jobs()

	var completed atomic.Int32
	release := make(chan struct{})

	for i := 0; i < 3; i++ {
		jobs <- func(ctx context.Context) error {
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			completed.Add(1)
			return nil
		}
	}

	pool.Close()
	close(release)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := pool.Wait(ctx); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if got := completed.Load(); got != 3 {
		t.Fatalf("completed jobs = %d, want 3", got)
	}
}

func TestWorkerPoolWaitCancelsActiveJobsOnTimeout(t *testing.T) {
	pool := NewWorkerPool(1, 1)
	jobs := pool.Jobs()

	started := make(chan struct{})
	jobErr := make(chan error, 1)

	jobs <- func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		jobErr <- ctx.Err()
		return ctx.Err()
	}

	pool.Close()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := pool.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait() error = %v, want deadline exceeded", err)
	}

	if err := <-jobErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("job error = %v, want context canceled", err)
	}
}
