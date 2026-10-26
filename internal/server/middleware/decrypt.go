package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/crypt"
)

// Decrypt returns a middleware that decrypts request bodies with the private
// key. A body that cannot be decrypted is rejected; requests without a body,
// such as GET, pass as they are.
func Decrypt(key *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
			_ = r.Body.Close()
			if err != nil {
				rejectBody(w, err)
				return
			}

			if len(body) > 0 {
				body, err = crypt.Decrypt(body, key)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			r.Header.Del("Content-Length")
			r.ContentLength = int64(len(body))
			next.ServeHTTP(w, r)
		})
	}
}
