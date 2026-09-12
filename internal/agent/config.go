package agent

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
)

const (
	defaultPollIntervalSeconds   = 2
	defaultReportIntervalSeconds = 10
)

type Config struct {
	ServerAddr string `env:"ADDRESS"`
	// Key signs every request body; an empty key sends requests unsigned.
	Key string `env:"KEY"`

	// Filled in GetConfig: the flags and the environment carry bare seconds,
	// which env.Parse cannot read into a duration.
	PollInterval   time.Duration `env:"-"`
	ReportInterval time.Duration `env:"-"`
}

func GetConfig() Config {
	cfg := Config{
		ServerAddr: "localhost:8080",
	}

	intervals := struct {
		Poll   int64 `env:"POLL_INTERVAL"`
		Report int64 `env:"REPORT_INTERVAL"`
	}{
		Poll:   defaultPollIntervalSeconds,
		Report: defaultReportIntervalSeconds,
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.Int64Var(&intervals.Poll, "p", intervals.Poll, "poll interval in seconds")
	flag.Int64Var(&intervals.Report, "r", intervals.Report, "report interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "key to sign requests with (unsigned if empty)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}
	if err := env.Parse(&intervals); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	cfg.PollInterval = secondsToDuration("poll interval", intervals.Poll)
	cfg.ReportInterval = secondsToDuration("report interval", intervals.Report)

	return cfg
}

// secondsToDuration also rejects a non-positive interval: unlike the server's
// store interval, zero has no meaning here and time.NewTicker would panic.
func secondsToDuration(name string, seconds int64) time.Duration {
	if seconds <= 0 {
		log.Fatalf("%s must be positive, got %d", name, seconds)
	}

	interval, err := time.ParseDuration(fmt.Sprintf("%ds", seconds))
	if err != nil {
		log.Fatalf("failed to parse %s: %v", name, err)
	}

	return interval
}
