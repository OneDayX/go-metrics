package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/longbridge/assert"
)

func TestListHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}
	tests := []struct {
		name        string
		haveMetrics []models.Metric
		want        want
	}{
		{
			name:        "success get metrics list",
			haveMetrics: []models.Metric{{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5)}},
			want: want{
				code: http.StatusOK,
				body: `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>Metrics List</title>
	</head>
	<body>
		<table>
				<tr>
					<td>Alloc</td>
					<td>12.5</td>
				</tr>
		</table>
	</body>
</html>`,
			},
		},
		{
			name:        "empty list",
			haveMetrics: []models.Metric{},
			want: want{
				code: http.StatusOK,
				body: `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>Metrics List</title>
	</head>
	<body>
		<table>
		</table>
	</body>
</html>`,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			svc := service.NewMetricService(storage)

			for _, metric := range tc.haveMetrics {
				storage.Update(metric)
			}

			r := httptest.NewRequest(http.MethodGet, "/", nil)

			w := httptest.NewRecorder()

			ListHandler(svc)(w, r)

			assert.Equal(t, tc.want.code, w.Code)
			assert.EqualHTML(t, tc.want.body, w.Body.String())
		})
	}
}
