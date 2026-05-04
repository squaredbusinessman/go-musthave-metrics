package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbMaxConns              = int32(10)
	dbMinConns              = int32(1)
	dbMaxConnLifetime       = time.Hour
	dbMaxConnIdleTime       = 30 * time.Minute
	dbHealthCheckPeriod     = time.Minute
	dbMaxConnLifetimeJitter = 5 * time.Minute
	dbConnectTimeout        = 30 * time.Second
)

func newDBPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database DSN: %w", err)
	}

	poolConfig.MaxConns = dbMaxConns
	poolConfig.MinConns = dbMinConns
	poolConfig.MaxConnLifetime = dbMaxConnLifetime
	poolConfig.MaxConnIdleTime = dbMaxConnIdleTime
	poolConfig.HealthCheckPeriod = dbHealthCheckPeriod
	poolConfig.MaxConnLifetimeJitter = dbMaxConnLifetimeJitter

	connectCtx, cancel := context.WithTimeout(ctx, dbConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	return pool, nil
}
