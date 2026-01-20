package retry

import (
	"context"
	"time"
)

var backoffs = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

// Do запускает механизм ретраев для необходимых ошибок
// (подключение к серверу (сетевой/транспортный сбой) и
// ошибка класса 08 в постгре (Connection Exception),
// повторять до 3 раз с паузами 1s, 3s, 5s) по заранее фиксированному расписанию.
func Do(ctx context.Context, isRetryable func(error) bool, op func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	for attempt := 0; attempt <= len(backoffs); attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = op()
		if err == nil || !isRetryable(err) {
			return err
		}

		if attempt == len(backoffs) {
			break
		}

		if err := wait(ctx, backoffs[attempt]); err != nil {
			return err
		}
	}

	return err
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
