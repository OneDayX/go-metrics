package handler

import (
	"net/http"
	"strconv"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/go-chi/chi/v5"
)

type metricUpdater interface {
	Update(metric models.Metric) error
}

// UpdateHandler returns an HTTP handler that updates a metric from URL parameters.
// URL pattern: POST /update/{type}/{name}/{value}
func UpdateHandler(svc metricUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		if metricType == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var metric models.Metric

		switch models.MetricType(metricType) {
		case models.MetricTypeCounter:
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric = models.Metric{
				ID:    metricName,
				MType: models.MetricTypeCounter,
				Delta: &value,
			}

		case models.MetricTypeGauge:
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric = models.Metric{
				ID:    metricName,
				MType: models.MetricTypeGauge,
				Value: &value,
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := svc.Update(metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
