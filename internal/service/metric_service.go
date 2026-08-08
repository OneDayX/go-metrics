package service

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"runtime"

	"github.com/OneDayX/go-metrics/internal/models"
)

type storager interface {
	Update(metric models.Metric) error
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

func (s *MetricService) Fetch(ID string) (models.Metric, error) {
	return s.storage.Fetch(ID)
}

func (s *MetricService) FetchAll() []models.Metric {
	return s.storage.FetchAll()
}

func (m *MetricService) Collect() error {
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
		if err := m.Update(models.Metric{ID: g.name, MType: models.MetricTypeGauge, Value: models.Ptr(g.value)}); err != nil {
			return err
		}
	}

	if err := m.Update(models.Metric{ID: "RandomValue", MType: models.MetricTypeGauge, Value: models.Ptr(rand.Float64())}); err != nil {
		return err
	}
	if err := m.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}); err != nil {
		return err
	}

	return nil
}

func (m *MetricService) Send(host string) error {
	client := &http.Client{}

	metrics := m.storage.FetchAll()
	for _, metric := range metrics {

		// For counters, send only the delta accumulated since the last poll.
		if metric.MType == models.MetricTypeCounter {
			delta := *metric.Delta - m.lastPollCount
			m.lastPollCount = *metric.Delta
			metric.Delta = &delta
		}

		body, err := json.Marshal(metric)
		if err != nil {
			return err
		}

		url := "http://" + host + "/update"

		// Compress the request body with gzip.
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(body); err != nil {
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(buf.Bytes()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return errors.New("failed to send metric")
		}

		if err := resp.Body.Close(); err != nil {
			return err
		}
	}

	return nil
}
