package handler

import (
	"context"
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

func TestJSONUpdatesHandler(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "success batch",
			contentType: "application/json",
			body:        `[{"id":"Alloc","type":"gauge","value":1.5},{"id":"PollCount","type":"counter","delta":5}]`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "empty batch is accepted",
			contentType: "application/json",
			body:        `[]`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "invalid content type",
			contentType: "text/plain",
			body:        `[{"id":"Alloc","type":"gauge","value":1.5}]`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "invalid json body",
			contentType: "application/json",
			body:        `not a json`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "single object instead of a list",
			contentType: "application/json",
			body:        `{"id":"Alloc","type":"gauge","value":1.5}`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "unknown metric type in the batch",
			contentType: "application/json",
			body:        `[{"id":"Alloc","type":"gauge","value":1.5},{"id":"X","type":"histogram","value":1}]`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "gauge with nil value in the batch",
			contentType: "application/json",
			body:        `[{"id":"Alloc","type":"gauge"}]`,
			wantCode:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			svc := service.NewMetricService(storage)

			req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			h := NewHandler(nil)
			w := httptest.NewRecorder()

			h.UpdatesJSON(svc)(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestJSONUpdatesHandlerStoresMetrics(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	// PollCount appears twice on purpose: duplicates inside one batch must
	// accumulate just like separate requests would.
	body := `[
		{"id":"Alloc","type":"gauge","value":1.5},
		{"id":"PollCount","type":"counter","delta":2},
		{"id":"PollCount","type":"counter","delta":3}
	]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	h := NewHandler(nil)
	w := httptest.NewRecorder()

	h.UpdatesJSON(svc)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, svc.FetchAll(context.Background()), 2)

	gauge, err := svc.Fetch(context.Background(), "Alloc")
	require.NoError(t, err)
	require.NotNil(t, gauge.Value)
	assert.Equal(t, 1.5, *gauge.Value)

	counter, err := svc.Fetch(context.Background(), "PollCount")
	require.NoError(t, err)
	require.NotNil(t, counter.Delta)
	assert.Equal(t, int64(5), *counter.Delta, "duplicates inside one batch accumulate")
	assert.Equal(t, models.MetricTypeCounter, counter.MType)
}
