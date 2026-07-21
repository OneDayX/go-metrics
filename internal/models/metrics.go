package models

// MetricType represents the type of a metric.
type MetricType string

const (
	// MetricTypeGauge represents a gauge metric (float64).
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeCounter represents a counter metric (int64).
	MetricTypeCounter MetricType = "counter"
)

type Metric struct {
	ID    string     `json:"id"`
	MType MetricType `json:"type"`
	Delta *int64     `json:"delta,omitempty"`
	Value *float64   `json:"value,omitempty"`
	Hash  string     `json:"hash,omitempty"`
}
