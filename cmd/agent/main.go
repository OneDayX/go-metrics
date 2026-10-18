// Agent collects runtime and system metrics and sends them to the server. The
// flags and environment variables are described in internal/agent.Config.
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
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// printBuildInfo prints the build info to stdout, N/A for the values that
// were not set at build time.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", orNA(buildVersion))
	fmt.Printf("Build date: %s\n", orNA(buildDate))
	fmt.Printf("Build commit: %s\n", orNA(buildCommit))
}

func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func run() error {
	cfg := agent.GetConfig()

	// ctx is cancelled on SIGINT/SIGTERM, which stops every goroutine.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc := service.NewMetricService(repository.NewMemStorage())

	return agent.New(cfg, svc).Run(ctx)
}
