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
			// Class 40: the transaction was rolled back and can be repeated.
			name: "serialization failure",
			err:  &pgconn.PgError{Code: pgerrcode.SerializationFailure},
			want: true,
		},
		{
			name: "deadlock detected",
			err:  &pgconn.PgError{Code: pgerrcode.DeadlockDetected},
			want: true,
		},
		{
			// Also Class 40, but a retry would hit the same constraint.
			name: "integrity constraint violation is not retried",
			err:  &pgconn.PgError{Code: pgerrcode.TransactionIntegrityConstraintViolation},
			want: false,
		},
		{
			// The statement may have committed, and the upsert is not idempotent.
			name: "unknown completion is not retried",
			err:  &pgconn.PgError{Code: pgerrcode.StatementCompletionUnknown},
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
