package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

type metricUpdaterJSON interface {
	Update(ctx context.Context, metric models.Metric) error
}

// UpdateJSON returns the handler for POST /update. It saves one metric from
// the JSON body, for example {"id":"PollCount","type":"counter","delta":5}.
// The request needs Content-Type: application/json.
//
// Status codes: 200 if the metric is saved, 400 if the body is not valid JSON,
// the type is unknown or the metric has no value.
func (h *Handler) UpdateJSON(svc metricUpdaterJSON) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			h.log.Warn("invalid content type",
				zap.String("uri", r.RequestURI),
				zap.String("content_type", ct),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metric models.Metric
		if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
			h.log.Warn("failed to decode metric JSON",
				zap.String("uri", r.RequestURI),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
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

		if err := svc.Update(r.Context(), metric); err != nil {
			h.log.Error("failed to update metric",
				zap.String("uri", r.RequestURI),
				zap.String("name", metric.ID),
				zap.String("type", string(metric.MType)),
				zap.Error(err),
			)
			w.WriteHeader(statusForError(err))
			return
		}

		h.audit(r, metric)

		w.WriteHeader(http.StatusOK)
	}
}
