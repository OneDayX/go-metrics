// Package service holds the metric logic on top of a storage. The server uses
// it to save and read metrics, the agent to collect and send them.
package service

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"runtime"
	"sync"

	"github.com/OneDayX/go-metrics/internal/compress"
	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/OneDayX/go-metrics/internal/httpclient"
	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/sign"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// ErrSendFailed means the server answered the agent, but not with 200.
var ErrSendFailed = errors.New("failed to send metrics")

type storager interface {
	Update(ctx context.Context, metric models.Metric) error
	UpdateBatch(ctx context.Context, metrics []models.Metric) error
	FetchAll(ctx context.Context) []models.Metric
	Fetch(ctx context.Context, name string) (models.Metric, error)
}

// MetricService works with metrics kept in a storage. It is safe for
// concurrent use if the storage is.
type MetricService struct {
	storage storager
	// client repeats requests that never reached the server.
	client *httpclient.Client

	// reported holds, per counter name, how much of its total the agent has
	// already sent; the server expects increments, not totals.
	mu       sync.Mutex
	reported map[string]int64
}

// NewMetricService returns a MetricService that keeps metrics in storage.
func NewMetricService(storage storager) *MetricService {
	return &MetricService{
		storage:  storage,
		client:   httpclient.New(nil),
		reported: make(map[string]int64),
	}
}

// Update saves one metric: a gauge replaces the old value, a counter is added
// to it.
func (s *MetricService) Update(ctx context.Context, metric models.Metric) error {
	return s.storage.Update(ctx, metric)
}

// UpdateBatch applies a batch of metrics in a single storage call, so that in
// synchronous mode the file is written once per request, not once per metric.
func (s *MetricService) UpdateBatch(ctx context.Context, metrics []models.Metric) error {
	return s.storage.UpdateBatch(ctx, metrics)
}

// Fetch returns the metric with the given ID. If there is none, the error
// wraps models.ErrMetricNotFound.
func (s *MetricService) Fetch(ctx context.Context, ID string) (models.Metric, error) {
	return s.storage.Fetch(ctx, ID)
}

// FetchAll returns all stored metrics in no particular order.
func (s *MetricService) FetchAll(ctx context.Context) []models.Metric {
	return s.storage.FetchAll(ctx)
}

// Collect stores the runtime.MemStats fields and RandomValue as gauges and
// adds one to the PollCount counter. The agent calls it on every poll.
func (s *MetricService) Collect(ctx context.Context) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	gauges := []struct {
		name  string
		value float64
	}{
		{"Alloc", float64(memStats.Alloc)},
		{"BuckHashSys", float64(memStats.BuckHashSys)},
		{"Frees", float64(memStats.Frees)},
		{"GCCPUFraction", memStats.GCCPUFraction},
		{"GCSys", float64(memStats.GCSys)},
		{"HeapAlloc", float64(memStats.HeapAlloc)},
		{"HeapIdle", float64(memStats.HeapIdle)},
		{"HeapInuse", float64(memStats.HeapInuse)},
		{"HeapObjects", float64(memStats.HeapObjects)},
		{"HeapReleased", float64(memStats.HeapReleased)},
		{"HeapSys", float64(memStats.HeapSys)},
		{"LastGC", float64(memStats.LastGC)},
		{"Lookups", float64(memStats.Lookups)},
		{"MCacheInuse", float64(memStats.MCacheInuse)},
		{"MCacheSys", float64(memStats.MCacheSys)},
		{"MSpanInuse", float64(memStats.MSpanInuse)},
		{"MSpanSys", float64(memStats.MSpanSys)},
		{"Mallocs", float64(memStats.Mallocs)},
		{"NextGC", float64(memStats.NextGC)},
		{"NumForcedGC", float64(memStats.NumForcedGC)},
		{"NumGC", float64(memStats.NumGC)},
		{"OtherSys", float64(memStats.OtherSys)},
		{"PauseTotalNs", float64(memStats.PauseTotalNs)},
		{"StackInuse", float64(memStats.StackInuse)},
		{"StackSys", float64(memStats.StackSys)},
		{"Sys", float64(memStats.Sys)},
		{"TotalAlloc", float64(memStats.TotalAlloc)},
	}

	for _, g := range gauges {
		if err := s.Update(ctx, models.Metric{ID: g.name, MType: models.MetricTypeGauge, Value: models.Ptr(g.value)}); err != nil {
			return err
		}
	}

	if err := s.Update(ctx, models.Metric{ID: "RandomValue", MType: models.MetricTypeGauge, Value: models.Ptr(rand.Float64())}); err != nil {
		return err
	}
	if err := s.Update(ctx, models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}); err != nil {
		return err
	}

	return nil
}

// CollectSystem stores what gopsutil reads from the OS: the memory and the
// load of every CPU the machine turns out to have.
func (s *MetricService) CollectSystem(ctx context.Context) error {
	memory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}

	metrics := []models.Metric{
		{ID: "TotalMemory", MType: models.MetricTypeGauge, Value: models.Ptr(float64(memory.Total))},
		{ID: "FreeMemory", MType: models.MetricTypeGauge, Value: models.Ptr(float64(memory.Free))},
	}

	// A zero interval measures the load since the previous call instead of
	// sleeping, so the poll takes no longer than the other ones.
	utilization, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return err
	}

	for i, percent := range utilization {
		metrics = append(metrics, models.Metric{
			ID:    fmt.Sprintf("CPUutilization%d", i+1),
			MType: models.MetricTypeGauge,
			Value: models.Ptr(percent),
		})
	}

	return s.storage.UpdateBatch(ctx, metrics)
}

// Batch takes everything collected so far and turns the counter totals into
// the increments the server expects. The metrics count as reported from here
// on; Send gives them back if the report does not make it.
func (s *MetricService) Batch(ctx context.Context) []models.Metric {
	metrics := s.storage.FetchAll(ctx)
	if len(metrics) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, metric := range metrics {
		if metric.MType == models.MetricTypeCounter && metric.Delta != nil {
			delta := *metric.Delta - s.reported[metric.ID]
			s.reported[metric.ID] = *metric.Delta

			metrics[i].Delta = &delta
		}
	}

	return metrics
}

// Send reports one batch to the server in a single gzipped request to
// POST /updates/. A non-empty realIP goes in the X-Real-IP header. A non-empty
// key signs the uncompressed body, a non-nil publicKey encrypts the compressed
// one. Workers call it side by side, as many at a time as the rate limit
// allows.
func (s *MetricService) Send(ctx context.Context, host, realIP, key string, publicKey *rsa.PublicKey, metrics []models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	if err := s.send(ctx, host, realIP, key, publicKey, metrics); err != nil {
		s.unreport(metrics)
		return err
	}

	return nil
}

func (s *MetricService) send(ctx context.Context, host, realIP, key string, publicKey *rsa.PublicKey, metrics []models.Metric) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	payload, err := compress.Encode(body)
	if err != nil {
		return err
	}

	// Encrypted bytes look random and do not compress, so compression goes first.
	if publicKey != nil {
		payload, err = crypt.Encrypt(payload, publicKey)
		if err != nil {
			return err
		}
	}

	// The server checks the signature after decompressing, so it covers the JSON.
	var signature string
	if key != "" {
		signature = sign.Sum(body, key)
	}

	return s.post(ctx, host, realIP, payload, signature)
}

// unreport takes the increments of a batch that never arrived back, so that
// the next batch counts them again and nothing is lost.
func (s *MetricService) unreport(metrics []models.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, metric := range metrics {
		if metric.MType == models.MetricTypeCounter && metric.Delta != nil {
			s.reported[metric.ID] -= *metric.Delta
		}
	}
}

// post sends one gzipped batch. Repeating a failed attempt is the client's job.
func (s *MetricService) post(ctx context.Context, host, realIP string, body []byte, signature string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+host+"/updates/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	if signature != "" {
		req.Header.Set(sign.Header, signature)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: server answered %d", ErrSendFailed, resp.StatusCode)
	}

	return nil
}
