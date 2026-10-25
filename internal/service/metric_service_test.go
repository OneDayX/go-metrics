package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/compress"
	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/sign"
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
			err := s.Update(context.Background(), tt.metric)
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
			err := m.Collect(context.Background())
			assert.NoError(t, err)

			_, err = m.storage.Fetch(context.Background(), tt.metricName)
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

	err := svc.Update(context.Background(), models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(42.5)})
	require.NoError(t, err)
	err = svc.Update(context.Background(), models.Metric{ID: "testCounter", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(10))})
	require.NoError(t, err)

	err = svc.Send(context.Background(), ts.Listener.Addr().String(), "", "", nil, svc.Batch(context.Background()))
	assert.NoError(t, err)
}

func TestMetricService_CollectSystem(t *testing.T) {
	svc := NewMetricService(repository.NewMemStorage())

	err := svc.CollectSystem(context.Background())
	require.NoError(t, err)

	for _, name := range []string{"TotalMemory", "FreeMemory", "CPUutilization1"} {
		_, err := svc.storage.Fetch(context.Background(), name)
		assert.NoError(t, err, name)
	}
}

func TestMetricService_Batch_CountsIncrements(t *testing.T) {
	svc := NewMetricService(repository.NewMemStorage())
	counter := models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}

	require.NoError(t, svc.Update(context.Background(), counter))
	require.NoError(t, svc.Update(context.Background(), counter))

	batch := svc.Batch(context.Background())
	require.Len(t, batch, 1)
	assert.Equal(t, int64(2), *batch[0].Delta)

	require.NoError(t, svc.Update(context.Background(), counter))

	batch = svc.Batch(context.Background())
	require.Len(t, batch, 1)
	assert.Equal(t, int64(1), *batch[0].Delta, "only what was collected since the last batch")
}

func TestMetricService_Batch_CountersAreIndependent(t *testing.T) {
	svc := NewMetricService(repository.NewMemStorage())

	require.NoError(t, svc.Update(context.Background(), models.Metric{ID: "A", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(10))}))
	require.NoError(t, svc.Update(context.Background(), models.Metric{ID: "B", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(3))}))
	svc.Batch(context.Background())

	require.NoError(t, svc.Update(context.Background(), models.Metric{ID: "A", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}))
	require.NoError(t, svc.Update(context.Background(), models.Metric{ID: "B", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(2))}))

	deltas := map[string]int64{}
	for _, metric := range svc.Batch(context.Background()) {
		deltas[metric.ID] = *metric.Delta
	}

	assert.Equal(t, map[string]int64{"A": 1, "B": 2}, deltas)
}

func TestMetricService_Send_ReturnsIncrementsOfAFailedBatch(t *testing.T) {
	svc := NewMetricService(repository.NewMemStorage())
	counter := models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}

	require.NoError(t, svc.Update(context.Background(), counter))

	err := svc.Send(context.Background(), "localhost:1", "", "", nil, svc.Batch(context.Background()))
	require.Error(t, err)

	batch := svc.Batch(context.Background())
	require.Len(t, batch, 1)
	assert.Equal(t, int64(1), *batch[0].Delta, "the failed increment is counted again")
}

func TestMetricService_Send_WithKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := compress.Decode(r.Body)
		require.NoError(t, err)
		assert.True(t, sign.Valid(body, "secret", r.Header.Get(sign.Header)))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := NewMetricService(repository.NewMemStorage())
	err := svc.Update(context.Background(), models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(42.5)})
	require.NoError(t, err)

	err = svc.Send(context.Background(), ts.Listener.Addr().String(), "", "secret", nil, svc.Batch(context.Background()))
	assert.NoError(t, err)
}

func TestMetricService_Send_WithRealIP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "192.168.1.10", r.Header.Get("X-Real-IP"))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := NewMetricService(repository.NewMemStorage())
	err := svc.Update(context.Background(), models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(42.5)})
	require.NoError(t, err)

	err = svc.Send(context.Background(), ts.Listener.Addr().String(), "192.168.1.10", "", nil, svc.Batch(context.Background()))
	assert.NoError(t, err)
}

func TestMetricService_Send_WithPublicKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encrypted, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		compressed, err := crypt.Decrypt(encrypted, key)
		require.NoError(t, err)
		body, err := compress.Decode(bytes.NewReader(compressed))
		require.NoError(t, err)
		assert.Contains(t, string(body), "testGauge")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := NewMetricService(repository.NewMemStorage())
	err = svc.Update(context.Background(), models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(42.5)})
	require.NoError(t, err)

	err = svc.Send(context.Background(), ts.Listener.Addr().String(), "", "", &key.PublicKey, svc.Batch(context.Background()))
	assert.NoError(t, err)
}

func TestMetricService_Send_ServerError(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	err := svc.Update(context.Background(), models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(1.0)})
	require.NoError(t, err)

	err = svc.Send(context.Background(), "localhost:1", "", "", nil, svc.Batch(context.Background()))
	assert.Error(t, err)
}

func TestMetricService_Send_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	storage := repository.NewMemStorage()
	svc := NewMetricService(storage)

	err := svc.Update(context.Background(), models.Metric{ID: "testGauge", MType: models.MetricTypeGauge, Value: models.Ptr(1.0)})
	require.NoError(t, err)

	err = svc.Send(context.Background(), ts.Listener.Addr().String(), "", "", nil, svc.Batch(context.Background()))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metric")
}
