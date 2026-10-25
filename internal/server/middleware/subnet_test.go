package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedSubnet(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name     string
		realIP   string
		wantCode int
	}{
		{name: "in the subnet", realIP: "192.168.1.10", wantCode: http.StatusOK},
		{name: "out of the subnet", realIP: "10.0.0.1", wantCode: http.StatusForbidden},
		{name: "no header", realIP: "", wantCode: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/updates", nil)
			if tt.realIP != "" {
				req.Header.Set(realIPHeader, tt.realIP)
			}

			rr := httptest.NewRecorder()
			TrustedSubnet(subnet)(next).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
		})
	}
}
