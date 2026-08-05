package middleware

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, log)
}

func LoggerFromContext(ctx context.Context) *zap.Logger {
	if log, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return log
	}
	return zap.NewNop()
}

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

// Logger returns a middleware that logs request and response details and
// injects the logger into the request context for downstream handlers.
func Logger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r.WithContext(WithLogger(r.Context(), log)))

			duration := time.Since(start)

			log.Info("request",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", duration),
				zap.Int("status", rw.responseData.status),
				zap.Int("size", rw.responseData.size),
			)
		})
	}
}
