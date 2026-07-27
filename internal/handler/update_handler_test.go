package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	type want struct {
		code int
		body string
	}

	testCases := []struct {
		name   string
		method string
		path   string
		want   want
	}{
		{
			name:   "valid counter update query",
			method: http.MethodPost,
			path:   "/update/counter/someMetric/1",
			want:   want{code: http.StatusOK, body: ""},
		},
		{
			name:   "valid gauge update query",
			method: http.MethodPost,
			path:   "/update/gauge/someMetric/1.2",
			want:   want{code: http.StatusOK, body: ""},
		},
		{
			name:   "empty query",
			method: http.MethodPost,
			path:   "/",
			want:   want{code: http.StatusNotFound, body: ""},
		},
		{
			name:   "bad counter update query value",
			method: http.MethodPost,
			path:   "/update/counter/someMetric/badValue",
			want:   want{code: http.StatusBadRequest, body: ""},
		},
		{
			name:   "bad gauge update query value",
			method: http.MethodPost,
			path:   "/update/gauge/someMetric/badValue",
			want:   want{code: http.StatusBadRequest, body: ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)

			chiCtx := chi.NewRouteContext()
			req := r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))

			params := strings.Split(tc.path, "/")
			if len(params) >= 5 {
				chiCtx.URLParams.Add("type", params[2])  // Extract type from path
				chiCtx.URLParams.Add("name", params[3])  // Extract name from path
				chiCtx.URLParams.Add("value", params[4]) // Extract value from path
			}

			w := httptest.NewRecorder()

			UpdateHandler(svc)(w, req)

			assert.Equal(t, tc.want.code, w.Code)
			assert.Equal(t, tc.want.body, w.Body.String())
		})
	}
}
