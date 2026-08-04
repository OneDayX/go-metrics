package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricService_Update(t *testing.T) {
	tests := []struct {
		name    string
		metric  models.Metric
		wantErr bool
	}{
		{
			name:    "success gauge update",
			metric:  models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(42.5)},
			wantErr: false,
		},
		{
			name:    "success counter update",
			metric:  models.Metric{ID: "testCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(10))},
			wantErr: false,
		},
		{
			name:    "error gauge with nil value",
			metric:  models.Metric{ID: "badGauge", MType: models.MetricTypeGauge, Value: nil},
			wantErr: true,
		},
		{
			name:    "error counter with nil delta",
			metric:  models.Metric{ID: "badCounter", MType: models.MetricTypeCounter, Delta: nil},
			wantErr: true,
		},
		{
			name:    "error unknown metric type",
			metric:  models.Metric{ID: "unknown", MType: "unknown", Value: models.Ptr(1.0)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			s := NewMetricService(storage)
			err := s.Update(tt.metric)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricService_Collect(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		wantErr    bool
	}{
		{
			name:       "success collect metric",
			metricName: "Alloc",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMetricService(repository.NewMemStorage())
			err := m.Collect()
			assert.NoError(t, err)

			_, err = m.storage.Fetch(tt.metricName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricService_Send_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	err := svc.Update(models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(42.5)})
	require.NoError(t, err)
	err = svc.Update(models.Metric{ID: "testCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(10))})
	require.NoError(t, err)

	err = svc.Send(ts.Listener.Addr().String())
	assert.NoError(t, err)
}

func TestMetricService_Send_ServerError(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	err := svc.Update(models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(1.0)})
	require.NoError(t, err)

	err = svc.Send("localhost:1")
	assert.Error(t, err)
}

func TestMetricService_Send_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	err := svc.Update(models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(1.0)})
	require.NoError(t, err)

	err = svc.Send(ts.Listener.Addr().String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metric")
}
