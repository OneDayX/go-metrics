// Agent collects runtime and system metrics and sends them to the server. The
// flags and environment variables are described in internal/agent.Config.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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

	// ctx is cancelled on SIGINT/SIGTERM, which stops every goroutine.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc := service.NewMetricService(repository.NewMemStorage())

	return agent.New(cfg, svc).Run(ctx)
}
