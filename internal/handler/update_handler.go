package handler

import (
	"net/http"
	"strconv"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type metricUpdater interface {
	Update(metric models.Metric) error
}

// Update returns an HTTP handler that updates a metric from URL parameters.
// URL pattern: POST /update/{type}/{name}/{value}
func (h *Handler) Update(svc metricUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		if metricType == "" {
			h.log.Warn("empty metric type",
				zap.String("uri", r.RequestURI),
			)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var metric models.Metric

		switch models.MetricType(metricType) {
		case models.MetricTypeCounter:
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				h.log.Warn("invalid counter value",
					zap.String("uri", r.RequestURI),
					zap.String("name", metricName),
					zap.String("value", metricValue),
					zap.Error(err),
				)
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
				h.log.Warn("invalid gauge value",
					zap.String("uri", r.RequestURI),
					zap.String("name", metricName),
					zap.String("value", metricValue),
					zap.Error(err),
				)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			metric = models.Metric{
				ID:    metricName,
				MType: models.MetricTypeGauge,
				Value: &value,
			}

		default:
			h.log.Warn("unsupported metric type",
				zap.String("uri", r.RequestURI),
				zap.String("type", metricType),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := svc.Update(metric); err != nil {
			h.log.Error("failed to update metric",
				zap.String("uri", r.RequestURI),
				zap.String("name", metricName),
				zap.String("type", metricType),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
