package audit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPObserver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, `{"ts":1,"metrics":["Alloc"],"ip_address":"127.0.0.1"}`, string(body))
	}))
	defer srv.Close()

	err := NewHTTPObserver(srv.URL).Notify(Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"})
	assert.NoError(t, err)
}

func TestHTTPObserverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := NewHTTPObserver(srv.URL).Notify(Event{TS: 1, Metrics: []string{"Alloc"}})
	assert.Error(t, err)
}
