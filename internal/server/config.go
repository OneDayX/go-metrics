// Package server holds the server config.
package server

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

const defaultStoreIntervalSeconds = 300

// Config holds the server settings. The json tags are the keys of the config
// file; the store interval is read from it in loadFile.
type Config struct {
	ServerAddr      string `env:"ADDRESS" json:"address"`              // host:port to listen on
	LogFile         string `env:"LOG_FILE" json:"log_file"`            // logs go to stdout if empty
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"store_file"` // file to save metrics to
	Restore         bool   `env:"RESTORE" json:"restore"`              // load metrics from the file on start
	// DatabaseDSN turns on the PostgreSQL storage; the file is not used then.
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`
	// Key checks request signatures and signs responses; empty turns both off.
	Key string `env:"KEY" json:"key"`
	// Audit is off if the value is empty.
	AuditFile string `env:"AUDIT_FILE" json:"audit_file"`
	AuditURL  string `env:"AUDIT_URL" json:"audit_url"`
	// PprofAddr serves net/http/pprof; profiling is off if the value is empty.
	PprofAddr string `env:"PPROF_ADDRESS" json:"pprof_address"`
	// CryptoKey is the path to the private key in PEM that decrypts request
	// bodies; decryption is off if the value is empty.
	CryptoKey string `env:"CRYPTO_KEY" json:"crypto_key"`
	// TrustedSubnet is a CIDR like "192.168.1.0/24": metrics are accepted only
	// from agents whose X-Real-IP is in it. Any agent is accepted if it is empty.
	TrustedSubnet string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`

	// Filled in GetConfig: the flag and STORE_INTERVAL carry bare seconds,
	// which env.Parse cannot read into a duration. Zero means synchronous saving.
	StoreInterval time.Duration `env:"-" json:"-"`
	// PrivateKey is loaded from CryptoKey in GetConfig; nil if CryptoKey is empty.
	PrivateKey *rsa.PrivateKey `env:"-" json:"-"`
	// TrustedNet is parsed from TrustedSubnet in GetConfig; nil if it is empty.
	TrustedNet *net.IPNet `env:"-" json:"-"`
}

// GetConfig reads the config from the JSON file, the command line flags and
// the environment variables; a flag wins over the file and a variable wins
// over its flag. The file is set with -c/-config or CONFIG. It stops the
// program if a value is invalid.
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

	// The file is read before the flags are registered: its values become the
	// flag defaults, so a flag on the command line wins over the file.
	configFile := config.FilePath(os.Args[1:])
	if configFile != "" {
		if err := loadFile(configFile, &cfg, &storeInterval.Seconds); err != nil {
			log.Fatalf("failed to read the config file %s: %v", configFile, err)
		}
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.IntVar(&storeInterval.Seconds, "i", storeInterval.Seconds, "store interval in seconds (0 for sync)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "path to file storage")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file on startup")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection string (postgres DSN)")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "key to check and sign bodies with (off if empty)")
	flag.StringVar(&cfg.LogFile, "l", cfg.LogFile, "path to log file (stdout if empty)")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "file to append audit events to (off if empty)")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL to post audit events to (off if empty)")
	flag.StringVar(&cfg.PprofAddr, "pprof", cfg.PprofAddr, "address for the pprof server (off if empty)")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the private key in PEM to decrypt bodies with (off if empty)")
	flag.StringVar(&cfg.TrustedSubnet, "t", cfg.TrustedSubnet, "trusted subnet of the agents in CIDR (any agent if empty)")
	// Already read by config.FilePath; registered last, so that flag.Parse
	// accepts them and the usage lists them.
	flag.StringVar(&configFile, "c", configFile, "path to the JSON config file")
	flag.StringVar(&configFile, "config", configFile, "path to the JSON config file (same as -c)")
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

	if cfg.TrustedSubnet != "" {
		_, subnet, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			log.Fatalf("failed to parse the trusted subnet %s: %v", cfg.TrustedSubnet, err)
		}
		cfg.TrustedNet = subnet
	}

	return cfg
}

// loadFile reads the JSON config file into cfg; keys missing from the file
// keep the values cfg already has. The file has the store interval as a
// duration like "1s", it is saved to storeSeconds in whole seconds, the way
// the flag carries it.
func loadFile(path string, cfg *Config, storeSeconds *int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}

	var file struct {
		StoreInterval string `json:"store_interval"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	if file.StoreInterval != "" {
		interval, err := time.ParseDuration(file.StoreInterval)
		if err != nil {
			return fmt.Errorf("store_interval: %w", err)
		}
		*storeSeconds = int(interval / time.Second)
	}

	return nil
}
