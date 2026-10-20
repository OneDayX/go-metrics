package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestAgentRun(t *testing.T) {
	var requests, inFlight, maxInFlight int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requests, 1)

		now := atomic.AddInt64(&inFlight, 1)
		for {
			max := atomic.LoadInt64(&maxInFlight)
			if now <= max || atomic.CompareAndSwapInt64(&maxInFlight, max, now) {
				break
			}
		}

		// Hold the request, so a second worker would overlap with this one.
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt64(&inFlight, -1)

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := Config{
		ServerAddr:     ts.Listener.Addr().String(),
		RateLimit:      2,
		PollInterval:   10 * time.Millisecond,
		ReportInterval: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err := New(cfg, service.NewMetricService(repository.NewMemStorage())).Run(ctx)

	assert.NoError(t, err)
	assert.Positive(t, atomic.LoadInt64(&requests))
	assert.LessOrEqual(t, atomic.LoadInt64(&maxInFlight), int64(cfg.RateLimit))
}

func TestAgentRunSendsOnShutdown(t *testing.T) {
	var requests int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requests, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// The report interval never comes, so only the shutdown can send a batch.
	cfg := Config{
		ServerAddr:     ts.Listener.Addr().String(),
		RateLimit:      1,
		PollInterval:   10 * time.Millisecond,
		ReportInterval: time.Hour,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := New(cfg, service.NewMetricService(repository.NewMemStorage())).Run(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&requests))
}
