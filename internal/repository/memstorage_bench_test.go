package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
)

// benchBatch returns n gauges and one counter, like a batch from the agent.
func benchBatch(n int) []models.Metric {
	metrics := make([]models.Metric, 0, n+1)
	for i := range n {
		metrics = append(metrics, models.Metric{
			ID:    fmt.Sprintf("Gauge%d", i),
			MType: models.MetricTypeGauge,
			Value: models.Ptr(float64(i)),
		})
	}
	metrics = append(metrics, models.Metric{
		ID:    "PollCount",
		MType: models.MetricTypeCounter,
		Delta: models.Ptr(int64(1)),
	})
	return metrics
}

func BenchmarkMemStorageUpdate(b *testing.B) {
	ms := NewMemStorage()
	metric := models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}

	for b.Loop() {
		_ = ms.Update(context.Background(), metric)
	}
}

func BenchmarkMemStorageUpdateBatch(b *testing.B) {
	ms := NewMemStorage()
	batch := benchBatch(30)

	for b.Loop() {
		_ = ms.UpdateBatch(context.Background(), batch)
	}
}

func BenchmarkMemStorageFetchAll(b *testing.B) {
	ms := NewMemStorage()
	_ = ms.UpdateBatch(context.Background(), benchBatch(30))

	for b.Loop() {
		ms.FetchAll(context.Background())
	}
}
