package handler

import (
	"net/http"
	"strconv"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/go-chi/chi/v5"
)

type metricFetcher interface {
	Fetch(name string) (models.Metric, error)
}

func GetHandler(svc metricFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mType := models.MetricType(chi.URLParam(r, "type"))
		metric, err := svc.Fetch(chi.URLParam(r, "name"))

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if mType != metric.MType {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch mType {
		case models.MetricTypeCounter:
			w.Write([]byte(strconv.FormatInt(int64(*metric.Delta), 10)))
		case models.MetricTypeGauge:
			w.Write([]byte(strconv.FormatFloat(float64(*metric.Value), 'f', -1, 64)))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}
