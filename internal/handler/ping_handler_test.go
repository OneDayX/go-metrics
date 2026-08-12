package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubPinger reports a fixed result, so the handler can be tested without a
// real database.
type stubPinger struct {
	err error
}

func (p stubPinger) Ping(ctx context.Context) error {
	return p.err
}

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name string
		db   pinger
		want int
	}{
		{
			name: "database is reachable",
			db:   stubPinger{err: nil},
			want: http.StatusOK,
		},
		{
			name: "database is unreachable",
			db:   stubPinger{err: errors.New("connection refused")},
			want: http.StatusInternalServerError,
		},
		{
			name: "database is not configured",
			db:   nil,
			want: http.StatusInternalServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(nil)

			r := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			h.Ping(tc.db)(w, r)

			assert.Equal(t, tc.want, w.Code)
		})
	}
}
