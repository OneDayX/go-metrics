package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPObserver sends events to a remote server with a POST request.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver returns an observer that posts events to url.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Notify posts the event as JSON. Any answer other than 200 is an error.
func (h *HTTPObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("audit server returned status %d", resp.StatusCode)
	}
	return nil
}
