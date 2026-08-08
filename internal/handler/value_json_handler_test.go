package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONValueHandler(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		haveMetric  models.Metric
		wantCode    int
	}{
		{
			name:        "success gauge value",
			contentType: "application/json",
			body:        `{"id":"LastGC","type":"gauge"}`,
			haveMetric:  models.Metric{ID: "LastGC", MType: models.MetricTypeGauge, Value: models.Ptr(float64(1744184459))},
			wantCode:    http.StatusOK,
		},
		{
			name:        "success counter value",
			contentType: "application/json",
			body:        `{"id":"PollCount","type":"counter"}`,
			haveMetric:  models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(10))},
			wantCode:    http.StatusOK,
		},
		{
			name:        "metric not found",
			contentType: "application/json",
			body:        `{"id":"Missing","type":"gauge"}`,
			haveMetric:  models.Metric{},
			wantCode:    http.StatusNotFound,
		},
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `{"id":"LastGC","type":"gauge"}`,
			haveMetric:  models.Metric{ID: "LastGC", MType: models.MetricTypeGauge, Value: models.Ptr(float64(1))},
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "invalid json body",
			contentType: "application/json",
			body:        `bad json`,
			haveMetric:  models.Metric{},
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "type mismatch",
			contentType: "application/json",
			body:        `{"id":"LastGC","type":"counter"}`,
			haveMetric:  models.Metric{ID: "LastGC", MType: models.MetricTypeGauge, Value: models.Ptr(float64(1))},
			wantCode:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			svc := service.NewMetricService(storage)

			if tt.haveMetric.ID != "" {
				require.NoError(t, storage.Update(tt.haveMetric))
			}

			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			h := NewHandler(nil)
			w := httptest.NewRecorder()

			h.ValueJSON(svc)(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestJSONValueHandlerReturnsStoredMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	haveMetric := models.Metric{ID: "LastGC", MType: models.MetricTypeGauge, Value: models.Ptr(float64(1744184459))}
	require.NoError(t, storage.Update(haveMetric))

	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"LastGC","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")

	h := NewHandler(nil)
	w := httptest.NewRecorder()

	h.ValueJSON(svc)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got models.Metric
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))

	assert.Equal(t, haveMetric.ID, got.ID)
	assert.Equal(t, haveMetric.MType, got.MType)
	require.NotNil(t, got.Value)
	assert.Equal(t, *haveMetric.Value, *got.Value)
}
