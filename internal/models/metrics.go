// Package models holds the metric type and errors shared by the server and the
// agent.
package models

// MetricType represents the type of a metric.
type MetricType string

const (
	// MetricTypeGauge represents a gauge metric (float64).
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeCounter represents a counter metric (int64).
	MetricTypeCounter MetricType = "counter"
)

// Metric is one gauge or counter. The same struct is sent as JSON and kept in
// the storages. A gauge has only Value set, a counter has only Delta set.
type Metric struct {
	ID    string     `json:"id"`              // metric name
	MType MetricType `json:"type"`            // gauge or counter
	Delta *int64     `json:"delta,omitempty"` // counter value
	Value *float64   `json:"value,omitempty"` // gauge value
	// Hash is not used: bodies are signed with the HashSHA256 header instead.
	Hash string `json:"hash,omitempty"`
}

// Ptr returns a pointer to the given value. Useful for creating pointer
// literals for Metric.Value and Metric.Delta fields.
func Ptr[T any](v T) *T {
	return &v
}
