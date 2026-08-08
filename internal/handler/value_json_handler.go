package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

type metricFetcherJSON interface {
	Fetch(name string) (models.Metric, error)
}

// Value returns an HTTP handler that fetches a single metric by JSON.
func (h *Handler) ValueJSON(svc metricFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type MetricRequest struct {
			id    string `json:"id"`
			mtype string `json:"type"`
		}
		var metricRequest MetricRequest

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&metricRequest); err != nil {
			h.log.Debug("cannot decode request JSON body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		metric, err := svc.Fetch(metricRequest.id)

		if err != nil {
			h.log.Warn("metric not found",
				zap.String("uri", r.RequestURI),
				zap.String("name", string(metricRequest.mtype)),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if models.MetricType(metricRequest.mtype) != metric.MType {
			h.log.Warn("metric type mismatch",
				zap.String("uri", r.RequestURI),
				zap.String("name", metricRequest.id),
				zap.String("requested_type", string(metricRequest.mtype)),
				zap.String("actual_type", string(metric.MType)),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if metric.MType != models.MetricTypeCounter && metric.MType != models.MetricTypeGauge {
			h.log.Warn("unsupported metric type",
				zap.String("uri", r.RequestURI),
				zap.String("name", metricRequest.id),
				zap.String("type", string(metricRequest.mtype)),
			)
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}
