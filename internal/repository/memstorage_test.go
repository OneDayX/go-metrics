package repository

import (
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
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

			if err := ms.Update(tt.metric); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				metric, err := ms.Fetch(tt.metric.ID)
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

			assert.Equal(t, tt.wantMetrics, ms.FetchAll())
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

			if metric, err := ms.Fetch(tt.args.name); tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, tt.wantMetric, metric)
			}
		})
	}
}
