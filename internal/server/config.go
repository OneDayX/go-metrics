package server

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr      string `env:"ADDRESS"`
	LogFile         string `env:"LOG_FILE"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func GetConfig() Config {
	cfg := Config{
		ServerAddr:      "localhost:8080",
		LogFile:         "",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         false,
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.StringVar(&cfg.LogFile, "l", cfg.LogFile, "path to log file (stdout if empty)")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "store interval in seconds (0 for sync)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "path to file storage")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file on startup")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	return cfg
}
