package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/sign"
)

type signingResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (s *signingResponseWriter) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
	}
}

func (s *signingResponseWriter) Write(p []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.body.Write(p)
}

// Hash returns a middleware that checks the HashSHA256 signature of a request
// body and signs the response body with the same key.
func Hash(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if signature := r.Header.Get(sign.Header); signature != "" {
				body, err := io.ReadAll(r.Body)
				r.Body.Close()
				if err != nil || !sign.Valid(body, key, signature) {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			sw := &signingResponseWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)

			if sw.status == 0 {
				sw.status = http.StatusOK
			}

			w.Header().Set(sign.Header, sign.Sum(sw.body.Bytes(), key))
			w.WriteHeader(sw.status)
			_, _ = w.Write(sw.body.Bytes())
		})
	}
}
