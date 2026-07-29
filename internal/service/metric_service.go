package service

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"reflect"
	"runtime"
	"strconv"

	"github.com/OneDayX/go-metrics/internal/models"
)

// runtimeMetricNames lists the runtime.MemStats gauge metric names collected by Collect().
var runtimeMetricNames = []string{
	"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
	"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
	"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
	"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
	"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
	"Sys", "TotalAlloc",
}

type storager interface {
	Update(metric models.Metric) error
	FetchAll() []models.Metric
	Fetch(name string) (models.Metric, error)
}

type MetricService struct {
	storage storager
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

func (m *MetricService) Collect() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	v := reflect.ValueOf(&memStats).Elem()
	for _, name := range runtimeMetricNames {
		field := v.FieldByName(name)
		var value float64
		switch field.Kind() {
		case reflect.Float64:
			value = field.Float()
		case reflect.Uint64, reflect.Uint32:
			value = float64(field.Uint())
		default:
			value = float64(field.Int())
		}
		m.Update(models.Metric{ID: name, MType: models.MetricTypeGauge, Value: models.Ptr(value)})
	}

	m.Update(models.Metric{ID: "RandomValue", MType: models.MetricTypeGauge, Value: models.Ptr(rand.Float64())})
	m.Update(models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))})
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
