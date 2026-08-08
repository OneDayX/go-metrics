package handler

import "go.uber.org/zap"

// Handler holds logger dependency.
type Handler struct {
	log *zap.Logger
}

// NewHandler returns a Handler with the given logger. It uses a nop logger
// if none is provided.
func NewHandler(log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{log: log}
}
