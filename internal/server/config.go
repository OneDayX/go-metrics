package server

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr string `env:"ADDRESS"`
	LogFile    string `env:"LOG_FILE"`
}

func GetConfig() Config {
	cfg := Config{
		ServerAddr: "localhost:8080",
		LogFile:    "",
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.StringVar(&cfg.LogFile, "l", cfg.LogFile, "path to log file (stdout if empty)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	return cfg
}
