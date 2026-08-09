package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/compress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipEncode(t *testing.T, data string) *bytes.Buffer {
	t.Helper()
	encoded, err := compress.Encode([]byte(data))
	require.NoError(t, err)
	return bytes.NewBuffer(encoded)
}

func TestGzipMiddlewareDecompressRequest(t *testing.T) {
	const body = `{"id":"test","type":"counter","delta":1}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(got))
		assert.Empty(t, r.Header.Get("Content-Encoding"))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/update", gzipEncode(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGzipMiddlewareDecompressRequestWithDeflate(t *testing.T) {
	const body = `{"id":"test","type":"counter","delta":1}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(got))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/update", gzipEncode(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip, deflate")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGzipMiddlewareInvalidGzipRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString("not gzip"))
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGzipMiddlewareCompressJSONResponse(t *testing.T) {
	const body = `{"id":"test","type":"counter","delta":1}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})

	req := httptest.NewRequest(http.MethodGet, "/value", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", rr.Header().Get("Vary"))

	decoded, err := compress.Decode(rr.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(decoded))
}

func TestGzipMiddlewareCompressHTMLResponse(t *testing.T) {
	const body = "<html><body>metrics</body></html>"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	decoded, err := compress.Decode(rr.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(decoded))
}

func TestGzipMiddlewareNoCompressionWithoutAcceptEncoding(t *testing.T) {
	const body = `{"id":"test","type":"counter","delta":1}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})

	req := httptest.NewRequest(http.MethodGet, "/value", nil)

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Content-Encoding"))
	assert.Equal(t, body, rr.Body.String())
}

func TestGzipMiddlewareNoCompressionOnErrorStatus(t *testing.T) {
	const body = `{"error":"not found"}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(body))
	})

	req := httptest.NewRequest(http.MethodGet, "/value", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Content-Encoding"))
	assert.Equal(t, body, rr.Body.String())
}

func TestAcceptsEncoding(t *testing.T) {
	tests := []struct {
		header   string
		encoding string
		want     bool
	}{
		{"gzip", "gzip", true},
		{"gzip, deflate", "gzip", true},
		{"deflate, gzip", "gzip", true},
		{"x-gzip", "gzip", true},
		{"gzip;q=0.8", "gzip", true},
		{"deflate", "gzip", false},
		{"notgzip", "gzip", false},
		{"", "gzip", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, acceptsEncoding(tt.header, tt.encoding), "header=%q", tt.header)
	}
}
