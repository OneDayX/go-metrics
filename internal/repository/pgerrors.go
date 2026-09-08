package repository

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// isRetriablePgError reports whether a postgres failure is worth repeating.
func isRetriablePgError(err error) bool {
	// Nobody waits for the answer any more. Checked first because
	// context.DeadlineExceeded also satisfies net.Error below.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return isUnavailableCode(pgErr.Code)
	}

	if pgconn.SafeToRetry(err) {
		return true
	}

	// The server dropped the connection in the middle of the query.
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

// Class 08 is the one the task names. The three codes
// next to it come from Class 57 and cover a restart of the database.
func isUnavailableCode(code string) bool {
	if pgerrcode.IsConnectionException(code) {
		return true
	}

	switch code {
	case pgerrcode.AdminShutdown, pgerrcode.CrashShutdown, pgerrcode.CannotConnectNow:
		return true
	default:
		return false
	}
}
