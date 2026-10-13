package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/compress"
)

// BenchmarkGzipMiddleware unpacks a gzipped request and packs a JSON response,
// like a POST /value/ from a client that accepts gzip.
func BenchmarkGzipMiddleware(b *testing.B) {
	body, err := compress.Encode([]byte(`{"id":"Alloc","type":"gauge"}`))
	if err != nil {
		b.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"Alloc","type":"gauge","value":2853240}`))
	})
	handler := GzipMiddleware(next)

	for b.Loop() {
		r := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(httptest.NewRecorder(), r)
	}
}
