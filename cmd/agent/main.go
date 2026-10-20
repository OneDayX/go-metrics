// Agent collects runtime and system metrics and sends them to the server. The
// flags, environment variables and config file keys are described in
// internal/agent.Config.
package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/OneDayX/go-metrics/internal/agent"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
)

// Build info, set at build time with
// -ldflags "-X main.buildVersion=... -X main.buildDate=... -X main.buildCommit=...".
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	printBuildInfo()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// printBuildInfo prints the build info to stdout.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func run() error {
	cfg := agent.GetConfig()

	// ctx is cancelled on SIGINT/SIGTERM/SIGQUIT; the agent then sends what it
	// has collected and exits.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	svc := service.NewMetricService(repository.NewMemStorage())

	return agent.New(cfg, svc).Run(ctx)
}
