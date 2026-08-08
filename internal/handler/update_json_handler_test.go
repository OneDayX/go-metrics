package handler

import (
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

func TestJSONUpdateHandler(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "success gauge update",
			contentType: "application/json",
			body:        `{"id":"LastGC","type":"gauge","value":1744184459}`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "success counter update",
			contentType: "application/json",
			body:        `{"id":"PollCount","type":"counter","delta":5}`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `{"id":"LastGC","type":"gauge","value":1}`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "invalid json body",
			contentType: "application/json",
			body:        `not a json`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "gauge with nil value",
			contentType: "application/json",
			body:        `{"id":"LastGC","type":"gauge"}`,
			wantCode:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			svc := service.NewMetricService(storage)

			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			h := NewHandler(nil)
			w := httptest.NewRecorder()

			h.UpdateJSON(svc)(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestJSONUpdateHandlerStoresMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	body := `{"id":"LastGC","type":"gauge","value":1744184459}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h := NewHandler(nil)
	w := httptest.NewRecorder()

	h.UpdateJSON(svc)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	metric, err := svc.Fetch("LastGC")
	require.NoError(t, err)
	assert.Equal(t, models.MetricTypeGauge, metric.MType)
	require.NotNil(t, metric.Value)
	assert.Equal(t, float64(1744184459), *metric.Value)
}
