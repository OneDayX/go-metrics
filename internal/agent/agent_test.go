package agent

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OneDayX/go-metrics/internal/grpcserver"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestAgentRunGRPC(t *testing.T) {
	// The server takes metrics only from the trusted subnet, so the batches
	// get through only with the agent's address in the metadata. The server
	// is on 127.0.0.1, so the agent finds its address there too.
	_, subnet, err := net.ParseCIDR("127.0.0.0/8")
	require.NoError(t, err)

	serverSvc := service.NewMetricService(repository.NewMemStorage())
	srv := grpcserver.New(serverSvc, nil, subnet)
	defer srv.Stop()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(listener) }()

	cfg := Config{
		GRPCAddr:       listener.Addr().String(),
		RateLimit:      1,
		PollInterval:   10 * time.Millisecond,
		ReportInterval: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = New(cfg, service.NewMetricService(repository.NewMemStorage())).Run(ctx)
	require.NoError(t, err)

	pollCount, err := serverSvc.Fetch(context.Background(), "PollCount")
	require.NoError(t, err)
	assert.Positive(t, *pollCount.Delta)
}
