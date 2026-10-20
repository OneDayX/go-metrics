package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptMiddleware(t *testing.T) {
	const body = `{"id":"test","type":"counter","delta":1}`

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	encrypted, err := crypt.Encrypt([]byte(body), &key.PublicKey)
	require.NoError(t, err)

	tests := []struct {
		name     string
		body     []byte
		wantCode int
		wantBody string
	}{
		{name: "encrypted body", body: encrypted, wantCode: http.StatusOK, wantBody: body},
		{name: "plain body", body: []byte(body), wantCode: http.StatusBadRequest},
		{name: "no body", body: nil, wantCode: http.StatusOK, wantBody: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.wantBody, string(got))
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(tt.body))
			rr := httptest.NewRecorder()
			Decrypt(key)(next).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
		})
	}
}
