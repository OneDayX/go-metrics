package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/OneDayX/go-metrics/internal/models"
)

// Persister saves the metrics of a MemStorage to a JSON file and loads them
// back on startup.
type Persister struct {
	mu       sync.Mutex
	storage  *MemStorage
	filePath string
	interval time.Duration
	ticker   *time.Ticker
}

// NewPersister returns a Persister for the given storage and file. Start saves
// the metrics every interval.
func NewPersister(storage *MemStorage, filePath string, interval time.Duration) *Persister {
	return &Persister{
		storage:  storage,
		filePath: filePath,
		interval: interval,
	}
}

// SaveMetrics writes all metrics to the file, replacing what was there.
func (p *Persister) SaveMetrics() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	metrics := p.storage.FetchAll(context.Background())
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(p.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics to file: %w", err)
	}

	return nil
}

// LoadMetrics reads metrics from the file into the storage. A missing file is
// not an error: there is just nothing to load.
func (p *Persister) LoadMetrics() error {
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read metrics from file: %w", err)
	}

	var metrics []models.Metric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	for _, metric := range metrics {
		if err := p.storage.Update(context.Background(), metric); err != nil {
			return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
		}
	}

	return nil
}

// Start saves the metrics every interval in a separate goroutine. It does
// nothing if the interval is zero.
func (p *Persister) Start() {
	// Not just "== 0": time.NewTicker panics on any non-positive interval.
	if p.interval <= 0 {
		return
	}

	p.ticker = time.NewTicker(p.interval)
	go func() {
		for range p.ticker.C {
			_ = p.SaveMetrics()
		}
	}()
}

// Stop stops the saving started by Start. It does not save the metrics itself.
func (p *Persister) Stop() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
}

// PersistentMemStorage is a wrapper around MemStorage that adds synchronous persistence.
// It intercepts Update() calls and immediately saves metrics to disk after each update.
type PersistentMemStorage struct {
	storage   *MemStorage
	persister *Persister
}

// NewPersistentMemStorage creates a new PersistentMemStorage wrapper.
// It should only be used when StoreInterval is 0 (synchronous save mode).
func NewPersistentMemStorage(storage *MemStorage, persister *Persister) *PersistentMemStorage {
	return &PersistentMemStorage{
		storage:   storage,
		persister: persister,
	}
}

// Update saves the metric in memory and writes the file right away.
func (pms *PersistentMemStorage) Update(ctx context.Context, metric models.Metric) error {
	err := pms.storage.Update(ctx, metric)
	if err != nil {
		return err
	}

	_ = pms.persister.SaveMetrics()
	return nil
}

// UpdateBatch applies the whole batch and writes the file once, instead of
// rewriting it after every metric.
func (pms *PersistentMemStorage) UpdateBatch(ctx context.Context, metrics []models.Metric) error {
	err := pms.storage.UpdateBatch(ctx, metrics)

	// Save even on a partial failure: metrics applied before the error are
	// already in memory, so the file must not fall behind.
	_ = pms.persister.SaveMetrics()

	return err
}

// FetchAll returns all metrics from memory.
func (pms *PersistentMemStorage) FetchAll(ctx context.Context) []models.Metric {
	return pms.storage.FetchAll(ctx)
}

// Fetch returns the metric with the given ID from memory.
func (pms *PersistentMemStorage) Fetch(ctx context.Context, ID string) (models.Metric, error) {
	return pms.storage.Fetch(ctx, ID)
}
