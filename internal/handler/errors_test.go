package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
)

func TestStatusForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "missing metric",
			err:  fmt.Errorf("%w: %q", models.ErrMetricNotFound, "Alloc"),
			want: http.StatusNotFound,
		},
		{
			name: "invalid metric",
			err:  fmt.Errorf("%w: gauge has no value", models.ErrInvalidMetric),
			want: http.StatusBadRequest,
		},
		{
			name: "unknown type",
			err:  fmt.Errorf("%w: %q", models.ErrUnknownMetricType, "histogram"),
			want: http.StatusBadRequest,
		},
		{
			name: "unknown error is a server problem",
			err:  errors.New("connection refused"),
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusForError(tt.err); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
