package main

import (
	"log"
	"time"

	"github.com/OneDayX/go-metrics/internal/agent"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := agent.DefaultConfig()
	cfg.ParseFlags()

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	// Poll ticker: collects runtime metrics at pollInterval.
	pollTicker := time.NewTicker(cfg.PollInterval)
	defer pollTicker.Stop()

	// Report ticker: sends metrics to the server at reportInterval.
	reportTicker := time.NewTicker(cfg.ReportInterval)
	defer reportTicker.Stop()

	// Perform an immediate first poll so we have data ready.
	svc.Collect()

	for {
		select {
		case <-pollTicker.C:
			svc.Collect()

		case <-reportTicker.C:
			if err := svc.Send(cfg.ServerAddr); err != nil {
				log.Printf("error sending metrics: %v", err)
			}
		}
	}
}
