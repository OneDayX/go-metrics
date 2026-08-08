package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

type metricUpdaterJSON interface {
	Update(metric models.Metric) error
}

// Update returns an HTTP handler that updates a metric from URL parameters.
// URL pattern: POST /update
func (h *Handler) UpdateJSON(svc metricUpdaterJSON) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var metric models.Metric
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&metric); err != nil {
			h.log.Debug("cannot decode request JSON body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if metric.MType != models.MetricTypeCounter && metric.MType != models.MetricTypeGauge {
			h.log.Warn("unsupported metric type",
				zap.String("uri", r.RequestURI),
				zap.String("type", string(metric.MType)),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := svc.Update(metric); err != nil {
			h.log.Error("failed to update metric",
				zap.String("uri", r.RequestURI),
				zap.String("name", metric.ID),
				zap.String("type", string(metric.MType)),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
