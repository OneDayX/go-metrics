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
	cfg := agent.GetConfig()

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	pollTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
	defer reportTicker.Stop()

	// Perform an immediate first poll so we have data ready.
	if err := svc.Collect(); err != nil {
		return err
	}

	for {
		select {
		case <-pollTicker.C:
			if err := svc.Collect(); err != nil {
				log.Printf("error collecting metrics: %v", err)
			}

		case <-reportTicker.C:
			if err := svc.Send(cfg.ServerAddr); err != nil {
				log.Printf("error sending metrics: %v", err)
			}
		}
	}
}
