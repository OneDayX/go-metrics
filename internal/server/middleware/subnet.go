package middleware

import (
	"net"
	"net/http"
)

// realIPHeader carries the address of the agent's host.
const realIPHeader = "X-Real-IP"

// TrustedSubnet returns a middleware that lets through only the requests whose
// X-Real-IP header holds an address from subnet. The rest, including requests
// without the header, get 403 Forbidden.
func TrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := net.ParseIP(r.Header.Get(realIPHeader))
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
