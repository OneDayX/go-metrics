package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/retry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const upsertQuery = `
INSERT INTO metrics (id, type, delta, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
	type = EXCLUDED.type,
	delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
	value = EXCLUDED.value`

// DBStorage keeps metrics in PostgreSQL.
type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(pool *pgxpool.Pool) *DBStorage {
	return &DBStorage{pool: pool}
}

func (ds *DBStorage) Update(ctx context.Context, metric models.Metric) error {
	if err := validateMetric(metric); err != nil {
		return err
	}

	err := retry.Do(func() error {
		_, err := ds.pool.Exec(ctx, upsertQuery,
			metric.ID, metric.MType, metric.Delta, metric.Value)
		return err
	}, isRetriablePgError)

	if err != nil {
		return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
	}

	return nil
}

// UpdateBatch applies all metrics in a single transaction, so a failing metric
// leaves the table untouched.
func (ds *DBStorage) UpdateBatch(ctx context.Context, metrics []models.Metric) error {
	// Validation does not touch the database, so it runs once, before any
	// attempt: a malformed metric will not become valid on a retry.
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return err
		}
	}

	// The whole transaction is repeated, not just the Exec: a dropped
	// connection can just as well break Begin or Commit.
	return retry.Do(func() error {
		return ds.updateBatchOnce(ctx, metrics)
	}, isRetriablePgError)
}

func (ds *DBStorage) updateBatchOnce(ctx context.Context, metrics []models.Metric) error {
	tx, err := ds.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, metric := range metrics {
		_, err := tx.Exec(ctx, upsertQuery, metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
		}
	}

	return tx.Commit(ctx)
}

func (ds *DBStorage) FetchAll(ctx context.Context) []models.Metric {
	var result []models.Metric

	err := retry.Do(func() error {
		var err error
		result, err = ds.fetchAllOnce(ctx)
		return err
	}, isRetriablePgError)

	if err != nil {
		return nil
	}

	return result
}

func (ds *DBStorage) fetchAllOnce(ctx context.Context) ([]models.Metric, error) {
	rows, err := ds.pool.Query(ctx,
		`SELECT id, type, delta, value FROM metrics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Metric, 0, 30)
	for rows.Next() {
		var metric models.Metric
		if err := rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value); err != nil {
			return nil, err
		}
		result = append(result, metric)
	}

	return result, rows.Err()
}

func (ds *DBStorage) Fetch(ctx context.Context, ID string) (models.Metric, error) {
	var metric models.Metric

	err := retry.Do(func() error {
		row := ds.pool.QueryRow(ctx,
			`SELECT id, type, delta, value FROM metrics WHERE id = $1`, ID)
		return row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
	}, isRetriablePgError)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.Metric{}, fmt.Errorf("%w: %q", models.ErrMetricNotFound, ID)
	}
	if err != nil {
		return models.Metric{}, err
	}

	return metric, nil
}
