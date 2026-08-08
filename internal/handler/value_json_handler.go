package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

// Value returns an HTTP handler that fetches a single metric by JSON.
func (h *Handler) ValueJSON(svc metricFetcher) http.HandlerFunc {
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

		stored, err := svc.Fetch(metric.ID)
		if err != nil {
			h.log.Warn("metric not found",
				zap.String("uri", r.RequestURI),
				zap.String("name", metric.ID),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if stored.MType != metric.MType {
			h.log.Warn("metric type mismatch",
				zap.String("uri", r.RequestURI),
				zap.String("name", metric.ID),
				zap.String("requested_type", string(metric.MType)),
				zap.String("actual_type", string(stored.MType)),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stored); err != nil {
			h.log.Error("failed to encode metric JSON",
				zap.String("uri", r.RequestURI),
				zap.String("name", metric.ID),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
