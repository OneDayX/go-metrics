package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsRetriablePgError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "class 08 connection failure",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			want: true,
		},
		{
			name: "plain error",
			err:  errors.New("something else"),
			want: false,
		},
		{
			name: "cancelled by the client",
			err:  fmt.Errorf("exec: %w", context.Canceled),
			want: false,
		},
		{
			// context.DeadlineExceeded satisfies net.Error, so without an
			// explicit check it would look like a temporary network fault.
			name: "request timed out",
			err:  fmt.Errorf("exec: %w", context.DeadlineExceeded),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetriablePgError(tt.err); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
