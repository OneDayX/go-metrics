package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_Update(t *testing.T) {
	type fields struct {
		metrics map[string]models.Metric
	}
	tests := []struct {
		name       string
		fields     fields
		metric     models.Metric
		wantMetric models.Metric
		wantErr    bool
	}{
		{
			name:       "update existing gauge",
			fields:     fields{metrics: map[string]models.Metric{"Alloc": {ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.0)}}},
			metric:     models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			wantMetric: models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			wantErr:    false,
		},
		{
			name:       "create new gauge",
			fields:     fields{metrics: map[string]models.Metric{}},
			metric:     models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			wantMetric: models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			wantErr:    false,
		},
		{
			name:       "update existing counter",
			fields:     fields{metrics: map[string]models.Metric{"PollCounter": {ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}}},
			metric:     models.Metric{ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(2))},
			wantMetric: models.Metric{ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(3))},
			wantErr:    false,
		},
		{
			name:       "create new counter",
			fields:     fields{metrics: map[string]models.Metric{}},
			metric:     models.Metric{ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(2))},
			wantMetric: models.Metric{ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(2))},
			wantErr:    false,
		},
		{
			name:    "create invalid metric",
			fields:  fields{metrics: map[string]models.Metric{}},
			metric:  models.Metric{ID: "PollCounter", MType: "sometype", Delta: models.Ptr(int64(2))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := MemStorage{
				metrics: tt.fields.metrics,
			}

			if err := ms.Update(context.Background(), tt.metric); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				metric, err := ms.Fetch(context.Background(), tt.metric.ID)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMetric, metric)
			}
		})
	}
}

func TestMemStorage_FetchAll(t *testing.T) {
	type fields struct {
		metrics map[string]models.Metric
	}
	tests := []struct {
		name        string
		fields      fields
		wantMetrics []models.Metric
		wantErr     bool
	}{
		{
			name: "success fetch metrics",
			fields: fields{
				metrics: map[string]models.Metric{
					"Alloc":       {ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
					"PollCounter": {ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))},
				},
			},
			wantMetrics: []models.Metric{
				{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
				{ID: "PollCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := MemStorage{
				metrics: tt.fields.metrics,
			}

			assert.ElementsMatch(t, tt.wantMetrics, ms.FetchAll(context.Background()))
		})
	}
}

func TestMemStorage_Fetch(t *testing.T) {
	type args struct {
		name string
	}
	type fields struct {
		metrics map[string]models.Metric
	}
	tests := []struct {
		args       args
		name       string
		fields     fields
		wantMetric models.Metric
		wantErr    bool
	}{
		{
			name: "success fetch metric",
			args: args{name: "Alloc"},
			fields: fields{
				metrics: map[string]models.Metric{
					"Alloc": {ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
				},
			},
			wantMetric: models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			wantErr:    false,
		},
		{
			name: "fetch not existed metric",
			args: args{name: "SomeMetric"},
			fields: fields{
				metrics: map[string]models.Metric{
					"Alloc": {ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := MemStorage{
				metrics: tt.fields.metrics,
			}

			if metric, err := ms.Fetch(context.Background(), tt.args.name); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, tt.wantMetric, metric)
			}
		})
	}
}

func TestValidateMetric(t *testing.T) {
	tests := []struct {
		name    string
		metric  models.Metric
		wantErr bool
	}{
		{
			name:    "valid gauge",
			metric:  models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)},
			wantErr: false,
		},
		{
			name:    "valid counter",
			metric:  models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))},
			wantErr: false,
		},
		{
			name:    "gauge without value",
			metric:  models.Metric{ID: "Alloc", MType: models.MetricTypeGauge},
			wantErr: true,
		},
		{
			name:    "counter without delta",
			metric:  models.Metric{ID: "PollCount", MType: models.MetricTypeCounter},
			wantErr: true,
		},
		{
			name:    "gauge with delta instead of value",
			metric:  models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Delta: models.Ptr(int64(1))},
			wantErr: true,
		},
		{
			name:    "unknown type",
			metric:  models.Metric{ID: "Alloc", MType: "histogram", Value: models.Ptr(1.5)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateMetric(tt.metric); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMemStorage_ConcurrentAccess reads and writes the map from several
// goroutines, the way the server does. Meaningful under -race.
func TestMemStorage_ConcurrentAccess(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	const goroutines = 8
	const iterations = 200

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range iterations {
				_ = ms.Update(ctx, models.Metric{
					ID:    fmt.Sprintf("Gauge%d", g),
					MType: models.MetricTypeGauge,
					Value: models.Ptr(float64(i)),
				})
				_ = ms.Update(ctx, models.Metric{
					ID:    "PollCount",
					MType: models.MetricTypeCounter,
					Delta: models.Ptr(int64(1)),
				})
				_, _ = ms.Fetch(ctx, "PollCount")
				_ = ms.FetchAll(ctx)
				_ = ms.UpdateBatch(ctx, []models.Metric{
					{ID: "Batched", MType: models.MetricTypeGauge, Value: models.Ptr(float64(i))},
				})
			}
		}(g)
	}
	wg.Wait()

	counter, err := ms.Fetch(ctx, "PollCount")
	require.NoError(t, err)
	assert.Equal(t, int64(goroutines*iterations), *counter.Delta,
		"every increment must be counted exactly once")
}

// The name limit mirrors the varchar(255) column.
func TestValidateMetric_NameLength(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "at the limit", id: strings.Repeat("a", maxMetricIDRunes), wantErr: false},
		{name: "one over the limit", id: strings.Repeat("a", maxMetricIDRunes+1), wantErr: true},
		{
			// 255 Cyrillic characters are 510 bytes.
			name:    "multi-byte name at the limit",
			id:      strings.Repeat("я", maxMetricIDRunes),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetric(models.Metric{ID: tt.id, MType: models.MetricTypeGauge, Value: models.Ptr(1.0)})

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, models.ErrInvalidMetric, "must map to 400, not 500")
		})
	}
}

// Both backends must reject the same name.
func TestStorages_RejectTheSameOverlongName(t *testing.T) {
	ctx := context.Background()
	metric := models.Metric{
		ID:    strings.Repeat("a", maxMetricIDRunes+1),
		MType: models.MetricTypeGauge,
		Value: models.Ptr(1.0),
	}

	err := NewMemStorage().Update(ctx, metric)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidMetric, "memory storage")

	ds := newTestStorage(t) // skips unless TEST_DATABASE_DSN is set
	err = ds.Update(ctx, metric)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidMetric, "database storage")
}
