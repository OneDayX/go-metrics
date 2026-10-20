// Package server holds the server config.
package server

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/caarlos0/env/v11"
)

const defaultStoreIntervalSeconds = 300

// Config holds the server settings.
type Config struct {
	ServerAddr      string `env:"ADDRESS"`           // host:port to listen on
	LogFile         string `env:"LOG_FILE"`          // logs go to stdout if empty
	FileStoragePath string `env:"FILE_STORAGE_PATH"` // file to save metrics to
	Restore         bool   `env:"RESTORE"`           // load metrics from the file on start
	// DatabaseDSN turns on the PostgreSQL storage; the file is not used then.
	DatabaseDSN string `env:"DATABASE_DSN"`
	// Key checks request signatures and signs responses; empty turns both off.
	Key string `env:"KEY"`
	// Audit is off if the value is empty.
	AuditFile string `env:"AUDIT_FILE"`
	AuditURL  string `env:"AUDIT_URL"`
	// PprofAddr serves net/http/pprof; profiling is off if the value is empty.
	PprofAddr string `env:"PPROF_ADDRESS"`
	// CryptoKey is the path to the private key in PEM that decrypts request
	// bodies; decryption is off if the value is empty.
	CryptoKey string `env:"CRYPTO_KEY"`

	// Filled in GetConfig: the flag and STORE_INTERVAL carry bare seconds,
	// which env.Parse cannot read into a duration. Zero means synchronous saving.
	StoreInterval time.Duration `env:"-"`
	// PrivateKey is loaded from CryptoKey in GetConfig; nil if CryptoKey is empty.
	PrivateKey *rsa.PrivateKey `env:"-"`
}

// GetConfig reads the config from the command line flags and the environment
// variables; a variable wins over its flag. It stops the program if a value is
// invalid.
func GetConfig() Config {
	cfg := Config{
		ServerAddr:      "localhost:8080",
		LogFile:         "",
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         false,
		DatabaseDSN:     "",
	}

	storeInterval := struct {
		Seconds int `env:"STORE_INTERVAL"`
	}{Seconds: defaultStoreIntervalSeconds}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.StringVar(&cfg.LogFile, "l", cfg.LogFile, "path to log file (stdout if empty)")
	flag.IntVar(&storeInterval.Seconds, "i", storeInterval.Seconds, "store interval in seconds (0 for sync)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "path to file storage")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file on startup")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection string (postgres DSN)")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "key to check and sign bodies with (off if empty)")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "file to append audit events to (off if empty)")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL to post audit events to (off if empty)")
	flag.StringVar(&cfg.PprofAddr, "pprof", cfg.PprofAddr, "address for the pprof server (off if empty)")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the private key in PEM to decrypt bodies with (off if empty)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}
	if err := env.Parse(&storeInterval); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	// A negative interval would panic in time.NewTicker.
	if storeInterval.Seconds < 0 {
		log.Fatalf("store interval must not be negative, got %d", storeInterval.Seconds)
	}

	interval, err := time.ParseDuration(fmt.Sprintf("%ds", storeInterval.Seconds))
	if err != nil {
		log.Fatalf("failed to parse store interval: %v", err)
	}
	cfg.StoreInterval = interval

	if cfg.CryptoKey != "" {
		key, err := crypt.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			log.Fatalf("failed to load the private key from %s: %v", cfg.CryptoKey, err)
		}
		cfg.PrivateKey = key
	}

	return cfg
}
