// Package server holds the server config.
package server

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/OneDayX/go-metrics/internal/config"
	"github.com/OneDayX/go-metrics/internal/crypt"
	"github.com/caarlos0/env/v11"
)

const defaultStoreIntervalSeconds = 300

// Config holds the server settings. The json tags are the keys of the config
// file; the store interval is read from it by config.LoadFile.
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
	// GRPCAddr is the host:port of the gRPC server, which runs next to the
	// HTTP one; gRPC is off if the value is empty.
	GRPCAddr string `env:"GRPC_ADDRESS" json:"grpc_address"`

	// Filled in Parse: the flag and STORE_INTERVAL carry bare seconds, which
	// env.Parse cannot read into a duration. Zero means synchronous saving.
	StoreInterval time.Duration `env:"-" json:"-"`
	// PrivateKey is loaded from CryptoKey in Parse; nil if CryptoKey is empty.
	PrivateKey *rsa.PrivateKey `env:"-" json:"-"`
	// TrustedNet is parsed from TrustedSubnet in Parse; nil if it is empty.
	TrustedNet *net.IPNet `env:"-" json:"-"`
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
		ServerAddr:      "localhost:8080",
		LogFile:         "",
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         false,
		DatabaseDSN:     "",
	}

	storeInterval := struct {
		Seconds int64 `env:"STORE_INTERVAL"`
	}{Seconds: defaultStoreIntervalSeconds}

	// The file is read before the flags are registered: its values become the
	// flag defaults, so a flag on the command line wins over the file.
	configFile := config.FilePath(args)
	if configFile != "" {
		err := config.LoadFile(configFile, &cfg, map[string]*int64{"store_interval": &storeInterval.Seconds})
		if err != nil {
			return Config{}, fmt.Errorf("read the config file %s: %w", configFile, err)
		}
	}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	fs.Int64Var(&storeInterval.Seconds, "i", storeInterval.Seconds, "store interval in seconds (0 for sync)")
	fs.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "path to file storage")
	fs.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file on startup")
	fs.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection string (postgres DSN)")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "key to check and sign bodies with (off if empty)")
	fs.StringVar(&cfg.LogFile, "l", cfg.LogFile, "path to log file (stdout if empty)")
	fs.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "file to append audit events to (off if empty)")
	fs.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL to post audit events to (off if empty)")
	fs.StringVar(&cfg.PprofAddr, "pprof", cfg.PprofAddr, "address for the pprof server (off if empty)")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the private key in PEM to decrypt bodies with (off if empty)")
	fs.StringVar(&cfg.TrustedSubnet, "t", cfg.TrustedSubnet, "trusted subnet of the agents in CIDR (any agent if empty)")
	fs.StringVar(&cfg.GRPCAddr, "grpc", cfg.GRPCAddr, "gRPC server address (host:port, off if empty)")
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
	if err := env.Parse(&storeInterval); err != nil {
		return Config{}, fmt.Errorf("parse environment variables: %w", err)
	}

	// A negative interval would panic in time.NewTicker.
	if storeInterval.Seconds < 0 {
		return Config{}, fmt.Errorf("store interval must not be negative, got %d", storeInterval.Seconds)
	}
	cfg.StoreInterval = time.Duration(storeInterval.Seconds) * time.Second

	if cfg.CryptoKey != "" {
		key, err := crypt.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			return Config{}, fmt.Errorf("load the private key from %s: %w", cfg.CryptoKey, err)
		}
		cfg.PrivateKey = key
	}

	if cfg.TrustedSubnet != "" {
		_, subnet, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			return Config{}, fmt.Errorf("parse the trusted subnet %s: %w", cfg.TrustedSubnet, err)
		}
		cfg.TrustedNet = subnet
	}

	return cfg, nil
}
