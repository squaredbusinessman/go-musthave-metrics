-- +goose UP
CREATE TABLE IF NOT EXISTS gauges (
    metric_name text PRIMARY KEY,
    value double precision NOT NULL
);

CREATE TABLE IF NOT EXISTS counters (
    metric_name text PRIMARY KEY,
    value bigint NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS gauges;
DROP TABLE IF EXISTS counters;