package repository

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/OneDayX/go-metrics/internal/models"
)

func TestPersister_SaveAndLoad(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	storage.Update(models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(100.5)})
	storage.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(42))})

	persister := NewPersister(storage, tmpFile, 1)
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
	persister2 := NewPersister(storage2, tmpFile, 1)
	if err := persister2.LoadMetrics(); err != nil {
		t.Fatalf("LoadMetrics failed: %v", err)
	}

	if len(storage2.FetchAll()) != 2 {
		t.Errorf("Expected 2 metrics after load, got %d", len(storage2.FetchAll()))
	}
}

func TestPersister_PeriodicSave(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"

	storage := NewMemStorage()
	storage.Update(models.Metric{ID: "TestMetric", MType: models.MetricTypeGauge, Value: models.Ptr(123.0)})

	persister := NewPersister(storage, tmpFile, 1)
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

	pms.Update(models.Metric{ID: "Gauge1", MType: models.MetricTypeGauge, Value: models.Ptr(50.5)})

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
