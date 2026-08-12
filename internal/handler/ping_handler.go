package handler

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const pingTimeout = 3 * time.Second

type pinger interface {
	Ping(ctx context.Context) error
}

// Ping returns an HTTP handler that checks the database connection.
// It answers 200 OK when the database is reachable and 500 otherwise.
func (h *Handler) Ping(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			h.log.Error("ping requested but database is not configured",
				zap.String("uri", r.RequestURI),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
