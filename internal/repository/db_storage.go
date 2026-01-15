package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
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
	_, err := db.pool.Exec(ctx, qUpsertGauge, name, value.Value)
	return err
}

func (db *DBStorage) AddCounter(ctx context.Context, name string, value int64) error {
	_, err := db.pool.Exec(ctx, qUpsertCounter, name, value)
	return err
}

func (db *DBStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var v float64
	err := db.pool.QueryRow(ctx, qGetGauge, name).Scan(&v)
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
	err := db.pool.QueryRow(ctx, qGetCounter, name).Scan(&v)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return v, nil
}

func (db *DBStorage) Snapshot(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error) {
	gauges := make(map[string]models.Gauge)
	counters := make(map[string]models.Counter)

	rows, err := db.pool.Query(ctx, qSnapshotGauges)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			rows.Close()
			return nil, nil, err
		}
		gauges[name] = models.Gauge{Value: value}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()

	rows, err = db.pool.Query(ctx, qSnapshotCounters)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			rows.Close()
			return nil, nil, err
		}
		counters[name] = models.Counter{Value: value}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()

	return gauges, counters, nil
}
