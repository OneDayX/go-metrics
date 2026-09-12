package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"runtime"

	"github.com/OneDayX/go-metrics/internal/compress"
	"github.com/OneDayX/go-metrics/internal/httpclient"
	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/sign"
)

// ErrSendFailed means the server answered the agent, but not with 200.
var ErrSendFailed = errors.New("failed to send metrics")

type storager interface {
	Update(ctx context.Context, metric models.Metric) error
	UpdateBatch(ctx context.Context, metrics []models.Metric) error
	FetchAll(ctx context.Context) []models.Metric
	Fetch(ctx context.Context, name string) (models.Metric, error)
}

type MetricService struct {
	storage storager
	// client repeats requests that never reached the server.
	client        *httpclient.Client
	lastPollCount int64
}

func NewMetricService(storage storager) *MetricService {
	return &MetricService{
		storage: storage,
		client:  httpclient.New(nil),
	}
}

func (s *MetricService) Update(ctx context.Context, metric models.Metric) error {
	return s.storage.Update(ctx, metric)
}

// UpdateBatch applies a batch of metrics in a single storage call, so that in
// synchronous mode the file is written once per request, not once per metric.
func (s *MetricService) UpdateBatch(ctx context.Context, metrics []models.Metric) error {
	return s.storage.UpdateBatch(ctx, metrics)
}

func (s *MetricService) Fetch(ctx context.Context, ID string) (models.Metric, error) {
	return s.storage.Fetch(ctx, ID)
}

func (s *MetricService) FetchAll(ctx context.Context) []models.Metric {
	return s.storage.FetchAll(ctx)
}

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

// Send reports every collected metric to the server in a single gzipped
// request to POST /updates/. A non-empty key signs the uncompressed body.
func (s *MetricService) Send(ctx context.Context, host, key string) error {
	metrics := s.storage.FetchAll(ctx)

	if len(metrics) == 0 {
		return nil
	}

	reported := s.lastPollCount
	for i, metric := range metrics {
		if metric.MType == models.MetricTypeCounter && metric.Delta != nil {
			reported = *metric.Delta

			delta := *metric.Delta - s.lastPollCount
			metrics[i].Delta = &delta
		}
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	compressed, err := compress.Encode(body)
	if err != nil {
		return err
	}

	// The server checks the signature after decompressing, so it covers the JSON.
	var signature string
	if key != "" {
		signature = sign.Sum(body, key)
	}

	if err := s.post(ctx, host, compressed, signature); err != nil {
		return err
	}

	s.lastPollCount = reported

	return nil
}

// post sends one gzipped batch. Repeating a failed attempt is the client's job.
func (s *MetricService) post(ctx context.Context, host string, body []byte, signature string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+host+"/updates/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if signature != "" {
		req.Header.Set(sign.Header, signature)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: server answered %d", ErrSendFailed, resp.StatusCode)
	}

	return nil
}
