package repository

import (
	"errors"
	"fmt"

	"github.com/OneDayX/go-metrics/internal/models"
)

type MemStorage struct {
	metrics map[string]models.Metric
}

func (ms *MemStorage) Update(metric models.Metric) error {
	switch metric.MType {
	case models.MetricTypeGauge:
		if metric.Value == nil {
			return errors.New("invalid value type for gauge metric: nil value")
		}
		ms.metrics[metric.ID] = metric

	case models.MetricTypeCounter:
		if metric.Delta == nil {
			return errors.New("invalid value type for counter metric: nil delta")
		}
		if existing, ok := ms.metrics[metric.ID]; ok {
			if existing.Delta == nil {
				return errors.New("existing counter metric has nil delta")
			}
			// Accumulate into a new local variable to avoid mutating
			// the caller's pointer (side-effect bug).
			accumulated := *existing.Delta + *metric.Delta
			metric.Delta = &accumulated
		}
		ms.metrics[metric.ID] = metric

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metric),
	}
}
