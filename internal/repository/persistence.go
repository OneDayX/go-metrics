package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/OneDayX/go-metrics/internal/models"
)

type Persister struct {
	mu       sync.Mutex
	storage  *MemStorage
	filePath string
	interval time.Duration
	ticker   *time.Ticker
}

func NewPersister(storage *MemStorage, filePath string, intervalSeconds int) *Persister {
	return &Persister{
		storage:  storage,
		filePath: filePath,
		interval: time.Duration(intervalSeconds) * time.Second,
	}
}

func (p *Persister) SaveMetrics() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	metrics := p.storage.FetchAll()
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(p.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics to file: %w", err)
	}

	return nil
}

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
		if err := p.storage.Update(metric); err != nil {
			return fmt.Errorf("failed to update metric %s: %w", metric.ID, err)
		}
	}

	return nil
}

func (p *Persister) Start() {
	if p.interval == 0 {
		return
	}

	p.ticker = time.NewTicker(p.interval)
	go func() {
		for range p.ticker.C {
			_ = p.SaveMetrics()
		}
	}()
}

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

func (pms *PersistentMemStorage) Update(metric models.Metric) error {
	err := pms.storage.Update(metric)
	if err != nil {
		return err
	}

	_ = pms.persister.SaveMetrics()
	return nil
}

// UpdateBatch applies the whole batch and writes the file once, instead of
// rewriting it after every metric.
func (pms *PersistentMemStorage) UpdateBatch(metrics []models.Metric) error {
	err := pms.storage.UpdateBatch(metrics)

	// Save even on a partial failure: metrics applied before the error are
	// already in memory, so the file must not fall behind.
	_ = pms.persister.SaveMetrics()

	return err
}

func (pms *PersistentMemStorage) FetchAll() []models.Metric {
	return pms.storage.FetchAll()
}

func (pms *PersistentMemStorage) Fetch(ID string) (models.Metric, error) {
	return pms.storage.Fetch(ID)
}
