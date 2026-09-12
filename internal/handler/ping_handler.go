package handler

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const pingTimeout = 3 * time.Second

// Pinger checks that the database is alive. Exported so that main can hold it
// as an interface: a nil *database.DB inside one would not be nil.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Ping returns an HTTP handler that checks the database connection. Running
// without a database is a valid setup, so that also answers 200.
func (h *Handler) Ping(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			h.log.Error("database ping failed",
				zap.String("uri", r.RequestURI),
				zap.Error(err),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
