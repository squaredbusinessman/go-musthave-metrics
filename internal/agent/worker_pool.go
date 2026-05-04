package agent

import (
	"context"
	"sync"

	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

// Job - задача, которую выполняет worker агента.
type Job func(context.Context) error

// WorkerPool - управляемый пул воркеров агента.
type WorkerPool struct {
	jobs   chan Job
	jobCtx context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// NewWorkerPool - запускает пул воркеров для обработки задач.
func NewWorkerPool(n int, queueSize int) *WorkerPool {
	if n < 1 {
		n = 1
	}
	if queueSize < 1 {
		queueSize = 1
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		jobs:   make(chan Job, queueSize),
		jobCtx: jobCtx,
		cancel: cancel,
	}

	for i := 0; i < n; i++ {
		pool.wg.Add(1)
		go func(id int) {
			defer pool.wg.Done()
			for job := range pool.jobs {
				if job == nil {
					continue
				}

				if err := job(pool.jobCtx); err != nil {
					myLog.Log.Warn("job failed", zap.Int("worker", id), zap.Error(err))
				}
			}
		}(i)
	}

	return pool
}

// Jobs - возвращает очередь задач для постановки новых отправок.
func (p *WorkerPool) Jobs() chan<- Job {
	return p.jobs
}

// Close - перестает принимать новые задачи и дает воркерам дренировать очередь.
func (p *WorkerPool) Close() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		close(p.jobs)
	})
}

// Wait - ждет завершения воркеров. При таймауте отменяет активные задачи.
func (p *WorkerPool) Wait(ctx context.Context) error {
	if p == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.cancel()
		return nil
	case <-ctx.Done():
		p.cancel()
		<-done
		return ctx.Err()
	}
}
