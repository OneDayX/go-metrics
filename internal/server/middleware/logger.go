package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
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

			fields := []zap.Field{
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
			}

			// Read and log the request body for POST requests.
			if r.Method == http.MethodPost && r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					fields = append(fields, zap.String("body", string(body)))
					// Restore the body so handlers can still read it.
					r.Body = io.NopCloser(bytes.NewReader(body))
				} else {
					fields = append(fields, zap.String("body", ""))
				}
			}

			rw := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			fields = append(fields,
				zap.Duration("duration", duration),
				zap.Int("status", rw.responseData.status),
				zap.Int("size", rw.responseData.size),
			)

			log.Info("request", fields...)
		})
	}
}
