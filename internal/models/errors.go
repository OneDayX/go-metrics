package models

import "errors"

// Sentinel errors shared by the storages.
var (
	// ErrMetricNotFound means the storage holds no metric with this ID.
	ErrMetricNotFound = errors.New("metric not found")

	// ErrInvalidMetric means the metric misses the value its type requires:
	// a gauge without value or a counter without delta.
	ErrInvalidMetric = errors.New("invalid metric")

	// ErrUnknownMetricType means the type is neither gauge nor counter.
	ErrUnknownMetricType = errors.New("unknown metric type")
)
