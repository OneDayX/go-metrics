package agent

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/caarlos0/env/v11"
)

const (
	defaultPollIntervalSeconds   = 2
	defaultReportIntervalSeconds = 10
	defaultRateLimit             = 1
)

// Config holds the agent settings.
type Config struct {
	// ServerAddr is the host:port of the server.
	ServerAddr string `env:"ADDRESS"`
	// Key signs every request body; an empty key sends requests unsigned.
	Key string `env:"KEY"`
	// RateLimit is how many requests the agent may have in flight at once.
	RateLimit int `env:"RATE_LIMIT"`
	// CryptoKey is the path to the server's public key in PEM; request bodies
	// are sent unencrypted if it is empty.
	CryptoKey string `env:"CRYPTO_KEY"`

	// Filled in GetConfig: the flags and the environment carry bare seconds,
	// which env.Parse cannot read into a duration.
	PollInterval   time.Duration `env:"-"`
	ReportInterval time.Duration `env:"-"`
	// PublicKey is loaded from CryptoKey in GetConfig; nil if CryptoKey is empty.
	PublicKey *rsa.PublicKey `env:"-"`
}

// GetConfig reads the config from the command line flags and the environment
// variables; a variable wins over its flag. It stops the program if a value is
// invalid.
func GetConfig() Config {
	cfg := Config{
		ServerAddr: "localhost:8080",
		RateLimit:  defaultRateLimit,
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
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "how many requests may be sent at once")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the server's public key in PEM (unencrypted if empty)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}
	if err := env.Parse(&intervals); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	if cfg.RateLimit <= 0 {
		log.Fatalf("rate limit must be positive, got %d", cfg.RateLimit)
	}

	cfg.PollInterval = secondsToDuration("poll interval", intervals.Poll)
	cfg.ReportInterval = secondsToDuration("report interval", intervals.Report)

	if cfg.CryptoKey != "" {
		key, err := crypt.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			log.Fatalf("failed to load the public key from %s: %v", cfg.CryptoKey, err)
		}
		cfg.PublicKey = key
	}

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
