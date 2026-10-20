// Package agent collects metrics and reports them to the server.
package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/service"
)

// Agent runs the polling and the reporting apart from each other: two
// goroutines fill the storage, one builds batches out of it, and a pool of
// workers sends them. The pool size is the limit of requests in flight.
type Agent struct {
	cfg Config
	svc *service.MetricService
}

// New returns an Agent that collects metrics with svc and sends them as cfg
// says.
func New(cfg Config, svc *service.MetricService) *Agent {
	return &Agent{cfg: cfg, svc: svc}
}

// Run collects and sends metrics until ctx is cancelled. It returns an error
// only if the very first poll fails.
func (a *Agent) Run(ctx context.Context) error {
	// Poll once right away, so the first report has something to send.
	if err := a.svc.Collect(ctx); err != nil {
		return err
	}

	batches := make(chan []models.Metric)

	var wg sync.WaitGroup

	collectors := []func(context.Context) error{a.svc.Collect, a.svc.CollectSystem}
	for _, collect := range collectors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.poll(ctx, collect)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.report(ctx, batches)
	}()

	for range a.cfg.RateLimit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.send(ctx, batches)
		}()
	}

	wg.Wait()
	return nil
}

// poll collects metrics into the storage on every tick.
func (a *Agent) poll(ctx context.Context, collect func(context.Context) error) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := collect(ctx); err != nil {
				log.Printf("error collecting metrics: %v", err)
			}
		}
	}
}

// report hands a batch to the workers on every tick. Closing the channel is
// what tells the workers to stop.
func (a *Agent) report(ctx context.Context, batches chan<- []models.Metric) {
	defer close(batches)

	ticker := time.NewTicker(a.cfg.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			batch := a.svc.Batch(ctx)
			if len(batch) == 0 {
				continue
			}

			select {
			case batches <- batch:
			case <-ctx.Done():
				return
			}
		}
	}
}

// send is one worker of the pool: it sends the batches it is given, one at a
// time, until the reporter closes the channel.
func (a *Agent) send(ctx context.Context, batches <-chan []models.Metric) {
	for batch := range batches {
		if err := a.svc.Send(ctx, a.cfg.ServerAddr, a.cfg.Key, a.cfg.PublicKey, batch); err != nil {
			log.Printf("error sending metrics: %v", err)
		}
	}
}
