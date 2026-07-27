package service

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strconv"

	"github.com/OneDayX/go-metrics/internal/models"
)

// storager defines the storage contract required by the service.
type storager interface {
	Update(metric models.Metric) error
	FetchAll() []models.Metric
	Fetch(name string) (models.Metric, error)
}

// MetricService provides business logic for metrics operations.
type MetricService struct {
	storage storager
}

// NewMetricService creates a new MetricService with the given storage backend.
func NewMetricService(storage storager) *MetricService {
	return &MetricService{
		storage: storage,
	}
}

// Update processes and stores a metric. For counters, the delta is accumulated
// with any existing value. For gauges, the value is overwritten.
func (s *MetricService) Update(metric models.Metric) error {
	return s.storage.Update(metric)
}

func (m *MetricService) Collect() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.Update(models.Metric{ID: "Alloc", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.Alloc))})
	m.Update(models.Metric{ID: "BuckHashSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.BuckHashSys))})
	m.Update(models.Metric{ID: "Frees", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.Frees))})
	m.Update(models.Metric{ID: "GCCPUFraction", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.GCCPUFraction))})
	m.Update(models.Metric{ID: "GCSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.GCSys))})
	m.Update(models.Metric{ID: "HeapAlloc", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.HeapAlloc))})
	m.Update(models.Metric{ID: "HeapIdle", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.HeapIdle))})
	m.Update(models.Metric{ID: "HeapInuse", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.HeapInuse))})
	m.Update(models.Metric{ID: "HeapObjects", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.HeapObjects))})
	m.Update(models.Metric{ID: "HeapReleased", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.HeapReleased))})
	m.Update(models.Metric{ID: "HeapSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.HeapSys))})
	m.Update(models.Metric{ID: "LastGC", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.LastGC))})
	m.Update(models.Metric{ID: "Lookups", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.Lookups))})
	m.Update(models.Metric{ID: "MCacheInuse", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.MCacheInuse))})
	m.Update(models.Metric{ID: "MCacheSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.MCacheSys))})
	m.Update(models.Metric{ID: "MSpanInuse", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.MSpanInuse))})
	m.Update(models.Metric{ID: "MSpanSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.MSpanSys))})
	m.Update(models.Metric{ID: "Mallocs", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.Mallocs))})
	m.Update(models.Metric{ID: "NextGC", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.NextGC))})
	m.Update(models.Metric{ID: "NumForcedGC", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.NumForcedGC))})
	m.Update(models.Metric{ID: "NumGC", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.NumGC))})
	m.Update(models.Metric{ID: "OtherSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.OtherSys))})
	m.Update(models.Metric{ID: "PauseTotalNs", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.PauseTotalNs))})
	m.Update(models.Metric{ID: "StackInuse", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.StackInuse))})
	m.Update(models.Metric{ID: "StackSys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.StackSys))})
	m.Update(models.Metric{ID: "Sys", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.Sys))})
	m.Update(models.Metric{ID: "TotalAlloc", MType: models.MetricTypeGauge, Value: ptr(float64(memStats.TotalAlloc))})

	m.Update(models.Metric{ID: "RandomValue", MType: models.MetricTypeGauge, Value: ptr(rand.Float64())})
	m.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: ptr(int64(1))})
}

func (m *MetricService) Send(host string) error {
	client := &http.Client{}

	var url string

	metrics := m.storage.FetchAll()
	for _, metric := range metrics {

		switch metric.MType {
		case models.MetricTypeGauge:
			url = "http://" + host + "/update/gauge/" + metric.ID + "/" + strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		case models.MetricTypeCounter:
			url = "http://" + host + "/update/counter/" + metric.ID + "/" + strconv.FormatInt(*metric.Delta, 10)
		}

		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.New("failed to send metric")
		}
	}

	return nil
}
