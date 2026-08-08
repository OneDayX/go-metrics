package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddlewareDecompressRequest(t *testing.T) {
	// gzip-encode a JSON body
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte(`{"id":"test","type":"counter","delta":1}`))
	require.NoError(t, gz.Close())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"id":"test","type":"counter","delta":1}`, string(body))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/update", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGzipMiddlewareCompressJSONResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"test","type":"counter","delta":1}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/value", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rr, req)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	zr, err := gzip.NewReader(rr.Body)
	require.NoError(t, err)
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Equal(t, `{"id":"test","type":"counter","delta":1}`, string(decoded))
}
