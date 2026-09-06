package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

type metricsBatchUpdater interface {
	UpdateBatch(metrics []models.Metric) error
}

func (h *Handler) UpdatesJSON(svc metricsBatchUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			h.log.Warn("invalid content type",
				zap.String("uri", r.RequestURI),
				zap.String("content_type", ct),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metrics []models.Metric
		if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
			h.log.Warn("failed to decode metrics JSON",
				zap.String("uri", r.RequestURI),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// An empty batch is not an error, there is simply nothing to store.
		if len(metrics) == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}

		for _, metric := range metrics {
			if metric.MType != models.MetricTypeCounter && metric.MType != models.MetricTypeGauge {
				h.log.Warn("unsupported metric type",
					zap.String("uri", r.RequestURI),
					zap.String("name", metric.ID),
					zap.String("type", string(metric.MType)),
				)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		if err := svc.UpdateBatch(metrics); err != nil {
			h.log.Error("failed to update metrics batch",
				zap.String("uri", r.RequestURI),
				zap.Int("count", len(metrics)),
				zap.Error(err),
			)
			w.WriteHeader(statusForError(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
