package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/OneDayX/go-metrics/internal/httpclient"
)

// HTTPObserver sends events to a remote server with a POST request. A request
// that fails on the network is repeated.
type HTTPObserver struct {
	url    string
	client *httpclient.Client
}

// NewHTTPObserver returns an observer that posts events to url.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: httpclient.New(&http.Client{Timeout: 5 * time.Second}),
	}
}

// Notify posts the event as JSON. Any answer other than 200 is an error.
func (h *HTTPObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// With a bytes.Reader body the request can be sent again on a retry.
	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("audit server returned status %d", resp.StatusCode)
	}
	return nil
}

// Close does nothing: HTTPObserver keeps nothing open between events.
func (h *HTTPObserver) Close() error {
	return nil
}
