package repository

import (
	"fmt"
	"sync"

	"github.com/OneDayX/go-metrics/internal/models"
)

type MemStorage struct {
	mu      sync.RWMutex
	metrics map[string]models.Metric
}

func validateMetric(metric models.Metric) error {
	switch metric.MType {
	case models.MetricTypeGauge:
		if metric.Value == nil {
			return fmt.Errorf("%w: gauge %q has no value", models.ErrInvalidMetric, metric.ID)
		}
	case models.MetricTypeCounter:
		if metric.Delta == nil {
			return fmt.Errorf("%w: counter %q has no delta", models.ErrInvalidMetric, metric.ID)
		}
	default:
		return fmt.Errorf("%w: %q", models.ErrUnknownMetricType, metric.MType)
	}

	return nil
}

func (ms *MemStorage) Update(metric models.Metric) error {
	if err := validateMetric(metric); err != nil {
		return err
	}

	if metric.MType == models.MetricTypeCounter {
		if existing, ok := ms.metrics[metric.ID]; ok && existing.Delta != nil {
			accumulated := *existing.Delta + *metric.Delta
			metric.Delta = &accumulated
		}
	}

	ms.metrics[metric.ID] = metric
	return nil
}

// UpdateBatch applies several metrics in one call.
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
		return models.Metric{}, fmt.Errorf("%w: %q", models.ErrMetricNotFound, ID)
	}
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metric),
	}
}
