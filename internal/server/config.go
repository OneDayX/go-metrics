package server

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr string `env:"ADDRESS"`
}

func GetConfig() Config {
	cfg := Config{
		ServerAddr: "localhost:8080",
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	return cfg
}
