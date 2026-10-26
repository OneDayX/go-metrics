package agent

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/OneDayX/go-metrics/internal/config"
	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/caarlos0/env/v11"
)

const (
	defaultPollIntervalSeconds   = 2
	defaultReportIntervalSeconds = 10
	defaultRateLimit             = 1
)

// Config holds the agent settings. The json tags are the keys of the config
// file; the intervals are read from it by config.LoadFile.
type Config struct {
	// ServerAddr is the host:port of the server.
	ServerAddr string `env:"ADDRESS" json:"address"`
	// Key signs every request body; an empty key sends requests unsigned.
	Key string `env:"KEY" json:"key"`
	// RateLimit is how many requests the agent may have in flight at once.
	RateLimit int `env:"RATE_LIMIT" json:"rate_limit"`
	// CryptoKey is the path to the server's public key in PEM; request bodies
	// are sent unencrypted if it is empty.
	CryptoKey string `env:"CRYPTO_KEY" json:"crypto_key"`
	// GRPCAddr is the host:port of the server's gRPC; if it is set, the
	// batches go over gRPC instead of HTTP.
	GRPCAddr string `env:"GRPC_ADDRESS" json:"grpc_address"`

	// Filled in Parse: the flags and the environment carry bare seconds,
	// which env.Parse cannot read into a duration.
	PollInterval   time.Duration `env:"-" json:"-"`
	ReportInterval time.Duration `env:"-" json:"-"`
	// PublicKey is loaded from CryptoKey in Parse; nil if CryptoKey is empty.
	PublicKey *rsa.PublicKey `env:"-" json:"-"`
}

// GetConfig reads the config of this run: Parse with the command line
// arguments.
func GetConfig() (Config, error) {
	return Parse(os.Args[1:])
}

// Parse reads the config from the JSON file, the flags in args and the
// environment variables; a flag wins over the file and a variable wins over
// its flag. The file is set with -c/-config or CONFIG. It fails if a value is
// invalid.
func Parse(args []string) (Config, error) {
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

	// The file is read before the flags are registered: its values become the
	// flag defaults, so a flag on the command line wins over the file.
	configFile := config.FilePath(args)
	if configFile != "" {
		err := config.LoadFile(configFile, &cfg, map[string]*int64{
			"poll_interval":   &intervals.Poll,
			"report_interval": &intervals.Report,
		})
		if err != nil {
			return Config{}, fmt.Errorf("read the config file %s: %w", configFile, err)
		}
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	fs.Int64Var(&intervals.Poll, "p", intervals.Poll, "poll interval in seconds")
	fs.Int64Var(&intervals.Report, "r", intervals.Report, "report interval in seconds")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "key to sign requests with (unsigned if empty)")
	fs.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "how many requests may be sent at once")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the server's public key in PEM (unencrypted if empty)")
	fs.StringVar(&cfg.GRPCAddr, "grpc", cfg.GRPCAddr, "gRPC server address (host:port, HTTP if empty)")
	// Already read by config.FilePath; registered last, so that fs.Parse
	// accepts them and the usage lists them.
	fs.StringVar(&configFile, "c", configFile, "path to the JSON config file")
	fs.StringVar(&configFile, "config", configFile, "path to the JSON config file (same as -c)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment variables: %w", err)
	}
	if err := env.Parse(&intervals); err != nil {
		return Config{}, fmt.Errorf("parse environment variables: %w", err)
	}

	// Without workers nobody reads the batches, and the agent would hang on
	// the first one.
	if cfg.RateLimit <= 0 {
		return Config{}, fmt.Errorf("rate limit must be positive, got %d", cfg.RateLimit)
	}

	var err error
	if cfg.PollInterval, err = secondsToDuration("poll interval", intervals.Poll); err != nil {
		return Config{}, err
	}
	if cfg.ReportInterval, err = secondsToDuration("report interval", intervals.Report); err != nil {
		return Config{}, err
	}

	if cfg.CryptoKey != "" {
		cfg.PublicKey, err = crypt.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			return Config{}, fmt.Errorf("load the public key from %s: %w", cfg.CryptoKey, err)
		}
	}

	return cfg, nil
}

// secondsToDuration also rejects a non-positive interval: unlike the server's
// store interval, zero has no meaning here and time.NewTicker would panic.
func secondsToDuration(name string, seconds int64) (time.Duration, error) {
	if seconds <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %d", name, seconds)
	}

	return time.Duration(seconds) * time.Second, nil
}
