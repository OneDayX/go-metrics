package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OneDayX/go-metrics/internal/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The real policy would make these tests wait for seconds.
func TestMain(m *testing.M) {
	original := retry.Delays
	retry.Delays = []time.Duration{0, 0, 0}
	defer func() { retry.Delays = original }()

	m.Run()
}

func TestDo_RetriesUntilTheServerAnswers(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dropped mid-request, which the client sees as a network failure.
		if atomic.AddInt32(&attempts, 1) <= 2 {
			hijacked, _, err := w.(http.Hijacker).Hijack()
			require.NoError(t, err)
			hijacked.Close()
			return
		}

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "batch", string(body), "every attempt must carry the full body")

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader([]byte("batch")))
	require.NoError(t, err)

	resp, err := New(nil).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts), "two failures, then success")
}

func TestDo_DoesNotRetryAnAnsweredRequest(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader([]byte("batch")))
	require.NoError(t, err)

	resp, err := New(nil).Do(req)
	require.NoError(t, err, "a server answer is not a transport failure")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "the server is alive, so nothing is repeated")
}

func TestDo_GivesUpOnAnUnreachableServer(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:1", bytes.NewReader([]byte("batch")))
	require.NoError(t, err)

	resp, err := New(nil).Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	assert.Error(t, err)
}

// A body the client cannot rebuild must be sent once and never repeated.
func TestDo_DoesNotRepeatAnUnrewindableBody(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		hijacked, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		hijacked.Close()
	}))
	defer srv.Close()

	// http.NewRequest cannot rebuild a body from a plain io.Reader.
	req, err := http.NewRequest(http.MethodPost, srv.URL, io.LimitReader(bytes.NewReader([]byte("batch")), 5))
	require.NoError(t, err)
	require.Nil(t, req.GetBody, "the test needs a request that cannot be rewound")

	resp, err := New(nil).Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	assert.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "sent once, not repeated")
}
