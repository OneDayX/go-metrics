package server

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
)

const defaultStoreIntervalSeconds = 300

type Config struct {
	ServerAddr      string `env:"ADDRESS"`
	LogFile         string `env:"LOG_FILE"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`

	// Filled in GetConfig: the flag and STORE_INTERVAL carry bare seconds,
	// which env.Parse cannot read into a duration. Zero means synchronous saving.
	StoreInterval time.Duration `env:"-"`
}

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

	return cfg
}
