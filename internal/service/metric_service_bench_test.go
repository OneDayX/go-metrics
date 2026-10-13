package service

import (
	"context"
	"testing"

	"github.com/OneDayX/go-metrics/internal/repository"
)

// BenchmarkCollect measures one poll of the runtime metrics on the agent.
func BenchmarkCollect(b *testing.B) {
	svc := NewMetricService(repository.NewMemStorage())

	for b.Loop() {
		_ = svc.Collect(context.Background())
	}
}
