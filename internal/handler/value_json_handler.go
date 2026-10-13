package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

// ValueJSON returns the handler for POST /value. It reads the name and type
// from the JSON body, for example {"id":"Alloc","type":"gauge"}, and answers
// with the stored metric as JSON, with its value or delta filled in. The
// request needs Content-Type: application/json.
//
// Status codes: 200 with the metric, 404 if there is no such metric, 400 if
// the body is not valid JSON or the metric has another type.
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

		stored, err := svc.Fetch(r.Context(), metric.ID)
		if err != nil {
			h.log.Warn("failed to fetch metric",
				zap.String("uri", r.RequestURI),
				zap.String("name", metric.ID),
				zap.Error(err),
			)
			w.WriteHeader(statusForError(err))
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
