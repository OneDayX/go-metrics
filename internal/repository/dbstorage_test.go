package repository

import (
	"context"
	"os"
	"testing"

	"github.com/OneDayX/go-metrics/internal/database"
	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *DBStorage {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set, skipping database tests")
	}

	db, err := database.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	pool := db.Pool()

	_, err = pool.Exec(context.Background(), `TRUNCATE metrics`)
	require.NoError(t, err)

	return NewDBStorage(pool)
}

func TestDBStorage_UpdateGauge(t *testing.T) {
	ds := newTestStorage(t)

	require.NoError(t, ds.Update(models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)}))
	require.NoError(t, ds.Update(models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(9.9)}))

	metric, err := ds.Fetch("Alloc")
	require.NoError(t, err)

	assert.Equal(t, models.MetricTypeGauge, metric.MType)
	assert.Equal(t, 9.9, *metric.Value, "gauge must be overwritten, not accumulated")
	assert.Nil(t, metric.Delta, "delta stays empty for a gauge")
}

func TestDBStorage_UpdateCounter(t *testing.T) {
	ds := newTestStorage(t)

	require.NoError(t, ds.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(5))}))
	require.NoError(t, ds.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(7))}))

	metric, err := ds.Fetch("PollCount")
	require.NoError(t, err)

	assert.Equal(t, models.MetricTypeCounter, metric.MType)
	assert.Equal(t, int64(12), *metric.Delta, "counter must accumulate: 5 + 7")
	assert.Nil(t, metric.Value)
}

func TestDBStorage_GaugeThenCounter(t *testing.T) {
	ds := newTestStorage(t)

	require.NoError(t, ds.Update(models.Metric{ID: "X", MType: models.MetricTypeGauge, Value: models.Ptr(2.0)}))
	require.NoError(t, ds.Update(models.Metric{ID: "X", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(5))}))

	metric, err := ds.Fetch("X")
	require.NoError(t, err)

	require.NotNil(t, metric.Delta, "delta must not stay NULL")
	assert.Equal(t, int64(5), *metric.Delta)
}

func TestDBStorage_UpdateInvalid(t *testing.T) {
	ds := newTestStorage(t)

	assert.Error(t, ds.Update(models.Metric{ID: "Alloc", MType: models.MetricTypeGauge}))
	assert.Error(t, ds.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter}))
	assert.Error(t, ds.Update(models.Metric{ID: "Alloc", MType: "histogram", Value: models.Ptr(1.0)}))
}

func TestDBStorage_FetchMissing(t *testing.T) {
	ds := newTestStorage(t)

	_, err := ds.Fetch("NoSuchMetric")
	assert.Error(t, err)
}

func TestDBStorage_FetchAll(t *testing.T) {
	ds := newTestStorage(t)

	assert.Empty(t, ds.FetchAll(), "an empty table gives an empty list")

	require.NoError(t, ds.Update(models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)}))
	require.NoError(t, ds.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(3))}))

	assert.ElementsMatch(t, []models.Metric{
		{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)},
		{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(3))},
	}, ds.FetchAll())
}

func TestDBStorage_UpdateBatch(t *testing.T) {
	ds := newTestStorage(t)

	require.NoError(t, ds.UpdateBatch([]models.Metric{
		{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)},
		{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(2))},
		{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(3))},
	}))

	assert.Len(t, ds.FetchAll(), 2)

	metric, err := ds.Fetch("PollCount")
	require.NoError(t, err)
	assert.Equal(t, int64(5), *metric.Delta, "duplicates inside one batch accumulate too")
}

// The batch runs in a transaction, so a broken metric must roll back everything
// written before it.
func TestDBStorage_UpdateBatchRollback(t *testing.T) {
	ds := newTestStorage(t)

	err := ds.UpdateBatch([]models.Metric{
		{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)},
		{ID: "Broken", MType: models.MetricTypeGauge}, // no value
	})
	require.Error(t, err)

	assert.Empty(t, ds.FetchAll(), "nothing must be left after the rollback")
}
