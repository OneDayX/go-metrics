package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
)

// benchMetrics returns 30 gauges and one counter, like a batch from the agent.
func benchMetrics() []models.Metric {
	metrics := []models.Metric{{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}}
	for i := range 30 {
		metrics = append(metrics, models.Metric{
			ID:    fmt.Sprintf("Gauge%d", i),
			MType: models.MetricTypeGauge,
			Value: models.Ptr(float64(i)),
		})
	}
	return metrics
}

// benchService returns a service whose storage already holds benchMetrics.
func benchService(b *testing.B) *service.MetricService {
	b.Helper()
	svc := service.NewMetricService(repository.NewMemStorage())
	if err := svc.UpdateBatch(context.Background(), benchMetrics()); err != nil {
		b.Fatal(err)
	}
	return svc
}

func BenchmarkUpdatesJSON(b *testing.B) {
	body, err := json.Marshal(benchMetrics())
	if err != nil {
		b.Fatal(err)
	}
	handler := NewHandler(nil, nil).UpdatesJSON(benchService(b))

	for b.Loop() {
		r := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		handler(httptest.NewRecorder(), r)
	}
}

func BenchmarkValueJSON(b *testing.B) {
	handler := NewHandler(nil, nil).ValueJSON(benchService(b))

	for b.Loop() {
		r := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"Gauge1","type":"gauge"}`))
		r.Header.Set("Content-Type", "application/json")
		handler(httptest.NewRecorder(), r)
	}
}

func BenchmarkList(b *testing.B) {
	handler := NewHandler(nil, nil).List(benchService(b))

	for b.Loop() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler(httptest.NewRecorder(), r)
	}
}
