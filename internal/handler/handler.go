package handler

import (
	"github.com/OneDayX/go-metrics/internal/audit"
	"go.uber.org/zap"
)

type auditor interface {
	Publish(event audit.Event)
}

// Handler holds logger and auditor dependencies.
type Handler struct {
	log     *zap.Logger
	auditor auditor
}

// NewHandler returns a Handler with the given logger and auditor. It uses a
// nop logger if none is provided. If auditor is nil, audit is off.
func NewHandler(log *zap.Logger, auditor auditor) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{log: log, auditor: auditor}
}
