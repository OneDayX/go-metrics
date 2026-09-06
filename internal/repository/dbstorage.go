package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/OneDayX/go-metrics/internal/models"
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

func (ds *DBStorage) Update(metric models.Metric) error {
	if err := validateMetric(metric); err != nil {
		return err
	}

	_, err := ds.pool.Exec(context.Background(), upsertQuery,
		metric.ID, metric.MType, metric.Delta, metric.Value)
	if err != nil {
		return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
	}

	return nil
}

// UpdateBatch applies all metrics in a single transaction, so a failing metric
// leaves the table untouched.
func (ds *DBStorage) UpdateBatch(metrics []models.Metric) error {
	ctx := context.Background()

	tx, err := ds.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, upsertQuery, metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
		}
	}

	return tx.Commit(ctx)
}

func (ds *DBStorage) FetchAll() []models.Metric {
	rows, err := ds.pool.Query(context.Background(),
		`SELECT id, type, delta, value FROM metrics`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]models.Metric, 0, 30)
	for rows.Next() {
		var metric models.Metric
		if err := rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value); err != nil {
			return nil
		}
		result = append(result, metric)
	}

	if rows.Err() != nil {
		return nil
	}

	return result
}

func (ds *DBStorage) Fetch(ID string) (models.Metric, error) {
	var metric models.Metric

	row := ds.pool.QueryRow(context.Background(),
		`SELECT id, type, delta, value FROM metrics WHERE id = $1`, ID)

	err := row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Metric{}, errors.New("metric not found")
	}
	if err != nil {
		return models.Metric{}, err
	}

	return metric, nil
}
