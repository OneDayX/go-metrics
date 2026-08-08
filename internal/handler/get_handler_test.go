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
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}
	tests := []struct {
		name       string
		haveMetric models.Metric
		path       string
		want       want
	}{
		{
			name:       "success get existing metric",
			haveMetric: models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			path:       "/value/gauge/Alloc",
			want:       want{code: http.StatusOK, body: "12.5"},
		},
		{
			name:       "not found metric",
			haveMetric: models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)},
			path:       "/value/gauge/someMetric",
			want:       want{code: http.StatusNotFound, body: ""},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			svc := service.NewMetricService(storage)

			svc.Update(tc.haveMetric)

			r := httptest.NewRequest(http.MethodGet, tc.path, nil)

			chiCtx := chi.NewRouteContext()
			req := r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))

			params := strings.Split(tc.path, "/")

			chiCtx.URLParams.Add("type", params[2]) // Extract type from path
			chiCtx.URLParams.Add("name", params[3]) // Extract name from path

			h := NewHandler(nil)

			w := httptest.NewRecorder()

			h.Get(svc)(w, req)

			assert.Equal(t, tc.want.code, w.Code)
			assert.Equal(t, tc.want.body, w.Body.String())
		})
	}
}
