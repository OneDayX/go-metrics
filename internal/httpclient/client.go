// Package httpclient wraps *http.Client with a retry policy, so that callers
// make a plain HTTP call and know nothing about repeated attempts.
package httpclient

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/OneDayX/go-metrics/internal/retry"
)

// Client sends HTTP requests, repeating those that never reached the server.
type Client struct {
	client *http.Client
}

// New wraps the given client; a nil client means http.DefaultClient.
func New(client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}

	return &Client{client: client}
}

// Do sends the request, retrying while the server cannot be reached. An answer
// is returned as is: the server is alive and would answer the same way again.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// A body that cannot be rebuilt must not be sent twice.
	replayable := req.Body == nil || req.GetBody != nil

	resp, err := c.attempt(req)

	for _, delay := range retry.Delays {
		if err == nil || !replayable || !isNetworkError(err) {
			break
		}

		time.Sleep(delay)
		resp, err = c.attempt(req)
	}

	return resp, err
}

// attempt sends one copy of the request.
func (c *Client) attempt(req *http.Request) (*http.Response, error) {
	fresh, err := rewind(req)
	if err != nil {
		return nil, err
	}

	return c.client.Do(fresh)
}

// rewind copies the request with a fresh body: every attempt drains it.
func rewind(req *http.Request) (*http.Request, error) {
	if req.Body == nil || req.GetBody == nil {
		return req, nil
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}

	attempt := req.Clone(req.Context())
	attempt.Body = body

	return attempt, nil
}

// isNetworkError reports whether the request failed to reach the server at all.
func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
