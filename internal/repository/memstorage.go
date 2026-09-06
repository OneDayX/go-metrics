package repository

import (
	"errors"
	"fmt"

	"github.com/OneDayX/go-metrics/internal/models"
)

type MemStorage struct {
	metrics map[string]models.Metric
}

// validateMetric checks that a metric carries the value its type requires.
// Both MemStorage and DBStorage rely on it before storing anything.
func validateMetric(metric models.Metric) error {
	switch metric.MType {
	case models.MetricTypeGauge:
		if metric.Value == nil {
			return errors.New("invalid value type for gauge metric: nil value")
		}
	case models.MetricTypeCounter:
		if metric.Delta == nil {
			return errors.New("invalid value type for counter metric: nil delta")
		}
	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func (ms *MemStorage) Update(metric models.Metric) error {
	if err := validateMetric(metric); err != nil {
		return err
	}

	// Counters accumulate, gauges are simply overwritten.
	if metric.MType == models.MetricTypeCounter {
		if existing, ok := ms.metrics[metric.ID]; ok && existing.Delta != nil {
			accumulated := *existing.Delta + *metric.Delta
			metric.Delta = &accumulated
		}
	}

	ms.metrics[metric.ID] = metric
	return nil
}

// UpdateBatch applies several metrics in one call. On error the metrics
// applied before it stay in the storage.
func (ms *MemStorage) UpdateBatch(metrics []models.Metric) error {
	for _, metric := range metrics {
		if err := ms.Update(metric); err != nil {
			return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
		}
	}
	return nil
}

func (ms *MemStorage) FetchAll() []models.Metric {
	result := make([]models.Metric, 0, 30)
	for _, metric := range ms.metrics {
		result = append(result, metric)
	}
	return result
}

func (ms *MemStorage) Fetch(ID string) (models.Metric, error) {
	if value, ok := ms.metrics[ID]; ok {
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
