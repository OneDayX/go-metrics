// Package handler contains the HTTP handlers of the metrics server.
//
// Every method of Handler returns an http.HandlerFunc for one endpoint:
//
//	POST /update/{type}/{name}/{value}  Update: save one metric from the URL
//	GET  /value/{type}/{name}           Get: value of one metric as text
//	GET  /                              List: HTML page with all metrics
//	GET  /ping                          Ping: check the database connection
//	POST /update                        UpdateJSON: save one metric from JSON
//	POST /updates                       UpdatesJSON: save a batch of metrics from JSON
//	POST /value                         ValueJSON: one metric as JSON
//
// The metric type in the URL and in JSON is "gauge" or "counter". A new gauge
// value replaces the old one, a counter value is added to the old one.
package handler

import (
	"github.com/OneDayX/go-metrics/internal/audit"
	"go.uber.org/zap"
)

type auditor interface {
	Publish(event audit.Event)
}

// Handler builds the HTTP handlers of the server. It holds the logger and the
// auditor that all handlers share.
type Handler struct {
	log     *zap.Logger
	auditor auditor
}

// NewHandler returns a Handler with the given logger and auditor. It uses a
// nop logger if none is provided. If auditor is nil, audit is off; otherwise
// it gets an event after every successful update.
func NewHandler(log *zap.Logger, auditor auditor) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{log: log, auditor: auditor}
}
