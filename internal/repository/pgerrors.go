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
		return isRetriableCode(pgErr.Code)
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

// isRetriableCode reports whether a SQLSTATE is worth a second attempt.
// Class 08 is the one the task names. The three codes
// next to it come from Class 57 and cover a restart of the database.
// The last two are from Class 40: the transaction was rolled back and can be
// repeated. They are listed by name instead of pgerrcode.IsTransactionRollback,
// which also covers 40002 and 40003, where a retry does not help.
func isRetriableCode(code string) bool {
	if pgerrcode.IsConnectionException(code) {
		return true
	}

	switch code {
	case pgerrcode.AdminShutdown, pgerrcode.CrashShutdown, pgerrcode.CannotConnectNow,
		pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
		return true
	default:
		return false
	}
}
