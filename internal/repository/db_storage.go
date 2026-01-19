package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/apperr"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/retry"
)

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(pool *pgxpool.Pool) *DBStorage {
	return &DBStorage{
		pool: pool,
	}
}

const (
	qUpsertGauge = `
		INSERT INTO gauges (metric_name, value)
		VALUES ($1, $2)
		ON CONFLICT (metric_name)
		DO UPDATE SET value = EXCLUDED.value

	`
	qUpsertCounter = `
		INSERT INTO counters (metric_name, value)
		VALUES ($1, $2)
		ON CONFLICT (metric_name)
		DO UPDATE SET value = counters.value + EXCLUDED.value

	`
	qGetGauge   = `SELECT value FROM gauges WHERE metric_name = $1`
	qGetCounter = `SELECT value FROM counters WHERE metric_name = $1`

	qSnapshotGauges   = `SELECT metric_name, value FROM gauges`
	qSnapshotCounters = `SELECT metric_name, value FROM counters`
)

func (db *DBStorage) SetGauge(ctx context.Context, name string, value models.Gauge) error {
	return retry.Do(ctx, isRetryablePGErr, func() error {
		_, err := db.pool.Exec(ctx, qUpsertGauge, name, value.Value)
		return err
	})
}

func (db *DBStorage) AddCounter(ctx context.Context, name string, value int64) error {
	return retry.Do(ctx, isRetryablePGErr, func() error {
		_, err := db.pool.Exec(ctx, qUpsertCounter, name, value)
		return err
	})
}

func (db *DBStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var v float64
	err := retry.Do(ctx, isRetryablePGErr, func() error {
		return db.pool.QueryRow(ctx, qGetGauge, name).Scan(&v)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return v, nil
}

func (db *DBStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	var v int64
	err := retry.Do(ctx, isRetryablePGErr, func() error {
		return db.pool.QueryRow(ctx, qGetCounter, name).Scan(&v)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return v, nil
}

func (db *DBStorage) Snapshot(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error) {
	var gauges map[string]models.Gauge
	var counters map[string]models.Counter

	err := retry.Do(ctx, isRetryablePGErr, func() error {
		gauges = make(map[string]models.Gauge)
		counters = make(map[string]models.Counter)

		rows, err := db.pool.Query(ctx, qSnapshotGauges)
		if err != nil {
			return err
		}
		for rows.Next() {
			var name string
			var value float64
			if err := rows.Scan(&name, &value); err != nil {
				rows.Close()
				return err
			}
			gauges[name] = models.Gauge{Value: value}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		rows, err = db.pool.Query(ctx, qSnapshotCounters)
		if err != nil {
			return err
		}
		for rows.Next() {
			var name string
			var value int64
			if err := rows.Scan(&name, &value); err != nil {
				rows.Close()
				return err
			}
			counters[name] = models.Counter{Value: value}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return gauges, counters, nil
}

func (db *DBStorage) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	for _, m := range metrics {
		switch m.MType {
		case models.MetricTypeGauge:
			if m.Value == nil {
				return apperr.ErrBadMetricValue
			}
		case models.MetricTypeCounter:
			if m.Delta == nil {
				return apperr.ErrBadMetricValue
			}
		default:
			return apperr.ErrUnknownMetricType
		}
	}

	return retry.Do(ctx, isRetryablePGErr, func() error {
		tx, err := db.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		b := &pgx.Batch{}
		queued := 0

		for _, m := range metrics {
			switch m.MType {
			case models.MetricTypeGauge:
				b.Queue(qUpsertGauge, m.ID, *m.Value)
				queued++
			case models.MetricTypeCounter:
				b.Queue(qUpsertCounter, m.ID, *m.Delta)
				queued++
			}
		}

		br := tx.SendBatch(ctx, b)

		for i := 0; i < queued; i++ {
			if _, err = br.Exec(); err != nil {
				_ = br.Close()
				return err
			}
		}

		if err := br.Close(); err != nil {
			return err
		}

		return tx.Commit(ctx)
	})
}

func isRetryablePGErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "08")
	}

	return false
}
