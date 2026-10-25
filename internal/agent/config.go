package agent

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
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
// file; the intervals are read from it in loadFile.
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

	// Filled in GetConfig: the flags and the environment carry bare seconds,
	// which env.Parse cannot read into a duration.
	PollInterval   time.Duration `env:"-" json:"-"`
	ReportInterval time.Duration `env:"-" json:"-"`
	// PublicKey is loaded from CryptoKey in GetConfig; nil if CryptoKey is empty.
	PublicKey *rsa.PublicKey `env:"-" json:"-"`
	// RealIP is the address of the agent's host, sent in X-Real-IP or in the
	// x-real-ip metadata; found in GetConfig.
	RealIP string `env:"-" json:"-"`
}

// GetConfig reads the config from the JSON file, the command line flags and
// the environment variables; a flag wins over the file and a variable wins
// over its flag. The file is set with -c/-config or CONFIG. It stops the
// program if a value is invalid.
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

	// The file is read before the flags are registered: its values become the
	// flag defaults, so a flag on the command line wins over the file.
	configFile := config.FilePath(os.Args[1:])
	if configFile != "" {
		if err := loadFile(configFile, &cfg, &intervals.Poll, &intervals.Report); err != nil {
			log.Fatalf("failed to read the config file %s: %v", configFile, err)
		}
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.Int64Var(&intervals.Poll, "p", intervals.Poll, "poll interval in seconds")
	flag.Int64Var(&intervals.Report, "r", intervals.Report, "report interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "key to sign requests with (unsigned if empty)")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "how many requests may be sent at once")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the server's public key in PEM (unencrypted if empty)")
	flag.StringVar(&cfg.GRPCAddr, "grpc", cfg.GRPCAddr, "gRPC server address (host:port, HTTP if empty)")
	// Already read by config.FilePath; registered last, so that flag.Parse
	// accepts them and the usage lists them.
	flag.StringVar(&configFile, "c", configFile, "path to the JSON config file")
	flag.StringVar(&configFile, "config", configFile, "path to the JSON config file (same as -c)")
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

	// The route is checked to the server the batches go to.
	serverAddr := cfg.ServerAddr
	if cfg.GRPCAddr != "" {
		serverAddr = cfg.GRPCAddr
	}

	ip, err := hostIP(serverAddr)
	if err != nil {
		log.Fatalf("failed to find the host address: %v", err)
	}
	cfg.RealIP = ip

	return cfg
}

// hostIP returns the address of the interface the agent reaches the server
// through. Dialing UDP sends nothing, it only picks the route.
func hostIP(serverAddr string) (string, error) {
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

// loadFile reads the JSON config file into cfg; keys missing from the file
// keep the values cfg already has. The file has the intervals as durations
// like "1s", they are saved to pollSeconds and reportSeconds in whole
// seconds, the way the flags carry them.
func loadFile(path string, cfg *Config, pollSeconds, reportSeconds *int64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}

	var file struct {
		PollInterval   string `json:"poll_interval"`
		ReportInterval string `json:"report_interval"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	if err := setSeconds(pollSeconds, file.PollInterval); err != nil {
		return fmt.Errorf("poll_interval: %w", err)
	}
	if err := setSeconds(reportSeconds, file.ReportInterval); err != nil {
		return fmt.Errorf("report_interval: %w", err)
	}

	return nil
}

// setSeconds saves a duration like "1s" to dst in whole seconds; an empty
// value leaves dst as it is.
func setSeconds(dst *int64, value string) error {
	if value == "" {
		return nil
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*dst = int64(interval / time.Second)

	return nil
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
