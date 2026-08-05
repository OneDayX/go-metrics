package handler

import (
	"net/http"
	"strconv"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/server/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type metricFetcher interface {
	Fetch(name string) (models.Metric, error)
}

func GetHandler(svc metricFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.LoggerFromContext(r.Context())

		mType := models.MetricType(chi.URLParam(r, "type"))
		name := chi.URLParam(r, "name")
		metric, err := svc.Fetch(name)

		if err != nil {
			log.Warn("metric not found",
				zap.String("uri", r.RequestURI),
				zap.String("name", name),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if mType != metric.MType {
			log.Warn("metric type mismatch",
				zap.String("uri", r.RequestURI),
				zap.String("name", name),
				zap.String("requested_type", string(mType)),
				zap.String("actual_type", string(metric.MType)),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch mType {
		case models.MetricTypeCounter:
			w.Write([]byte(strconv.FormatInt(int64(*metric.Delta), 10)))
		case models.MetricTypeGauge:
			w.Write([]byte(strconv.FormatFloat(float64(*metric.Value), 'f', -1, 64)))
		default:
			log.Warn("unsupported metric type",
				zap.String("uri", r.RequestURI),
				zap.String("name", name),
				zap.String("type", string(mType)),
			)
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}
