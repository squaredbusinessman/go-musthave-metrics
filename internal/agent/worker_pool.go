package agent

import (
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

type Job func() error

func StartWorkers(n int, jobs <-chan Job) {
	if n < 1 {
		n = 1
	}

	for i := 0; i < n; i++ {
		go func(id int) {
			for job := range jobs {
				if job == nil {
					continue
				}

				if err := job(); err != nil {
					myLog.Log.Warn("job failed", zap.Int("worker", id), zap.Error(err))
				}
			}
		}(i)
	}
}
