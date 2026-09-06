package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"runtime"

	"github.com/OneDayX/go-metrics/internal/compress"
	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/retry"
)

// ErrSendFailed means the server answered the agent, but not with 200.
var ErrSendFailed = errors.New("failed to send metrics")

type storager interface {
	Update(metric models.Metric) error
	UpdateBatch(metrics []models.Metric) error
	FetchAll() []models.Metric
	Fetch(name string) (models.Metric, error)
}

type MetricService struct {
	storage       storager
	lastPollCount int64
}

func NewMetricService(storage storager) *MetricService {
	return &MetricService{
		storage: storage,
	}
}

func (s *MetricService) Update(metric models.Metric) error {
	return s.storage.Update(metric)
}

// UpdateBatch applies a batch of metrics in a single storage call, so that in
// synchronous mode the file is written once per request, not once per metric.
func (s *MetricService) UpdateBatch(metrics []models.Metric) error {
	return s.storage.UpdateBatch(metrics)
}

func (s *MetricService) Fetch(ID string) (models.Metric, error) {
	return s.storage.Fetch(ID)
}

func (s *MetricService) FetchAll() []models.Metric {
	return s.storage.FetchAll()
}

func (s *MetricService) Collect() error {
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
		if err := s.Update(models.Metric{ID: g.name, MType: models.MetricTypeGauge, Value: models.Ptr(g.value)}); err != nil {
			return err
		}
	}

	if err := s.Update(models.Metric{ID: "RandomValue", MType: models.MetricTypeGauge, Value: models.Ptr(rand.Float64())}); err != nil {
		return err
	}
	if err := s.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}); err != nil {
		return err
	}

	return nil
}

// Send reports every collected metric to the server in a single gzipped
// request to POST /updates/.
func (s *MetricService) Send(host string) error {
	metrics := s.storage.FetchAll()

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

	if err := retry.Do(func() error { return post(host, compressed) }, isRetriableSendError); err != nil {
		return err
	}

	s.lastPollCount = reported

	return nil
}

// post sends one gzipped batch. It builds a fresh request on every call,
// because a retried request cannot reuse a body reader that is already drained.
func post(host string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, "http://"+host+"/updates/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: server answered %d", ErrSendFailed, resp.StatusCode)
	}

	return nil
}

// isRetriableSendError reports whether the server was simply unreachable.
// A network failure is worth another try; an answer we did not like means the
// server is alive and would reject the same batch again.
func isRetriableSendError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
