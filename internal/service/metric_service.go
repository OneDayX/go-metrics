package service

import (
	"github.com/OneDayX/go-metrics/internal/models"
)

// storager defines the storage contract required by the service.
type storager interface {
	Update(metric models.Metric) error
}

// MetricService provides business logic for metrics operations.
type MetricService struct {
	storage storager
}

// NewMetricService creates a new MetricService with the given storage backend.
func NewMetricService(storage storager) *MetricService {
	return &MetricService{
		storage: storage,
	}
}

// Update processes and stores a metric. For counters, the delta is accumulated
// with any existing value. For gauges, the value is overwritten.
func (s *MetricService) Update(metric models.Metric) error {
	return s.storage.Update(metric)
}
