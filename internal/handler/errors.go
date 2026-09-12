package handler

import (
	"errors"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
)

// statusForError turns a storage error into an HTTP status by introspecting it
// with errors.Is.
func statusForError(err error) int {
	switch {
	case errors.Is(err, models.ErrMetricNotFound):
		return http.StatusNotFound

	case errors.Is(err, models.ErrInvalidMetric),
		errors.Is(err, models.ErrUnknownMetricType):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
