package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/sign"
	"github.com/stretchr/testify/assert"
)

func TestHashMiddleware(t *testing.T) {
	const key = "secret"
	const body = `{"id":"test","type":"counter","delta":1}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name     string
		hash     string
		wantCode int
	}{
		{name: "valid hash", hash: sign.Sum([]byte(body), key), wantCode: http.StatusOK},
		{name: "wrong hash", hash: sign.Sum([]byte(body), "other"), wantCode: http.StatusBadRequest},
		{name: "no hash", hash: "", wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBufferString(body))
			if tt.hash != "" {
				req.Header.Set(sign.Header, tt.hash)
			}

			rr := httptest.NewRecorder()
			Hash(key)(next).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode == http.StatusOK {
				assert.Equal(t, sign.Sum([]byte("ok"), key), rr.Header().Get(sign.Header))
			}
		})
	}
}
