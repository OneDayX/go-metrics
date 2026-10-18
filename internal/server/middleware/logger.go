package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	// maxBodySize is a safety cap, not a size a real request gets close to: a
	// batch of all the agent's metrics is a few kilobytes of JSON. A larger body
	// is rejected as a whole, never passed on cut short.
	maxBodySize = 1 << 20 // 1 MiB

	// maxLoggedBody is how much of the body goes to the log.
	maxLoggedBody = 4 << 10 // 4 KiB
)

type responseData struct {
	status int
	size   int
}

type responseWriter struct {
	http.ResponseWriter
	responseData responseData
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.responseData.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Set 200 OK status if the handler did not call WriteHeader explicitly.
	if rw.responseData.status == 0 {
		rw.responseData.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.responseData.size += n
	return n, err
}

// Logger returns a middleware that logs request and response details.
func Logger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w}

			fields := []zap.Field{
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
			}

			// Deferred, so that a rejected request is logged as well.
			defer func() {
				fields = append(fields,
					zap.Duration("duration", time.Since(start)),
					zap.Int("status", rw.responseData.status),
					zap.Int("size", rw.responseData.size),
				)
				log.Info("request", fields...)
			}()

			if r.Method == http.MethodPost && r.Body != nil {
				body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
				fields = append(fields, zap.ByteString("body", body[:min(len(body), maxLoggedBody)]))

				if err != nil {
					rejectBody(rw, err)
					return
				}

				// Restore the body so handlers can still read it.
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			next.ServeHTTP(rw, r)
		})
	}
}

// rejectBody answers a request whose body could not be read in full.
func rejectBody(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		status = http.StatusRequestEntityTooLarge
	}

	http.Error(w, http.StatusText(status), status)
}
