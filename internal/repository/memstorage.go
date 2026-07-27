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
			accumulated := *existing.Delta + *metric.Delta
			metric.Delta = &accumulated
		}
		ms.metrics[metric.ID] = metric

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func (ms MemStorage) FetchAll() []models.Metric {
	result := make([]models.Metric, 0, 30)
	for _, metric := range ms.metrics {
		result = append(result, metric)
	}
	return result
}

func (ms MemStorage) Fetch(name string) (models.Metric, error) {
	if value, ok := ms.metrics[name]; ok {
		return value, nil
	} else {
		return models.Metric{}, errors.New("metric not found")
	}
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metric),
	}
}
