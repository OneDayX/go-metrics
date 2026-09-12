package repository

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/OneDayX/go-metrics/internal/models"
)

func TestPersister_SaveAndLoad(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	storage.Update(context.Background(), models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(100.5)})
	storage.Update(context.Background(), models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(42))})

	persister := NewPersister(storage, tmpFile, time.Second)
	if err := persister.SaveMetrics(); err != nil {
		t.Fatalf("SaveMetrics failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var metrics []models.Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
	}

	storage2 := NewMemStorage()
	persister2 := NewPersister(storage2, tmpFile, time.Second)
	if err := persister2.LoadMetrics(); err != nil {
		t.Fatalf("LoadMetrics failed: %v", err)
	}

	if len(storage2.FetchAll(context.Background())) != 2 {
		t.Errorf("Expected 2 metrics after load, got %d", len(storage2.FetchAll(context.Background())))
	}
}

func TestPersister_PeriodicSave(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	storage.Update(context.Background(), models.Metric{ID: "TestMetric", MType: models.MetricTypeGauge, Value: models.Ptr(123.0)})

	persister := NewPersister(storage, tmpFile, time.Second)
	persister.Start()
	time.Sleep(1500 * time.Millisecond)
	persister.Stop()

	if _, err := os.Stat(tmpFile); err != nil {
		t.Errorf("File was not created: %v", err)
	}
}

func TestPersistentMemStorage_SyncSave(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	persister := NewPersister(storage, tmpFile, 0)
	pms := NewPersistentMemStorage(storage, persister)

	pms.Update(context.Background(), models.Metric{ID: "Gauge1", MType: models.MetricTypeGauge, Value: models.Ptr(50.5)})

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("File should exist after Update: %v", err)
	}

	var metrics []models.Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(metrics) != 1 || *metrics[0].Value != 50.5 {
		t.Error("Metric was not saved correctly")
	}
}

func TestPersistentMemStorage_UpdateBatch(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	persister := NewPersister(storage, tmpFile, 0)
	pms := NewPersistentMemStorage(storage, persister)

	batch := []models.Metric{
		{ID: "Gauge1", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)},
		{ID: "Gauge2", MType: models.MetricTypeGauge, Value: models.Ptr(2.5)},
		{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(7))},
	}
	if err := pms.UpdateBatch(context.Background(), batch); err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("File should exist after UpdateBatch: %v", err)
	}

	var metrics []models.Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics in file, got %d", len(metrics))
	}
}

func TestPersistentMemStorage_UpdateBatchInvalidMetric(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	persister := NewPersister(storage, tmpFile, 0)
	pms := NewPersistentMemStorage(storage, persister)

	batch := []models.Metric{
		{ID: "Gauge1", MType: models.MetricTypeGauge, Value: models.Ptr(1.5)},
		{ID: "Broken", MType: models.MetricTypeGauge}, // nil value
	}
	if err := pms.UpdateBatch(context.Background(), batch); err == nil {
		t.Fatal("Expected an error for a metric with nil value")
	}

	// The valid metric applied before the error must still reach the file.
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("File should exist after a partial batch: %v", err)
	}

	var metrics []models.Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(metrics) != 1 || metrics[0].ID != "Gauge1" {
		t.Errorf("Expected only Gauge1 in file, got %+v", metrics)
	}
}
