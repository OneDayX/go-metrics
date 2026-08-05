package agent

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	PollInterval   int64  `env:"POLL_INTERVAL"`
	ReportInterval int64  `env:"REPORT_INTERVAL"`
	ServerAddr     string `env:"ADDRESS"`
}

func GetConfig() Config {
	cfg := Config{
		PollInterval:   2,
		ReportInterval: 10,
		ServerAddr:     "localhost:8080",
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.Int64Var(&cfg.PollInterval, "p", cfg.PollInterval, "poll interval in seconds")
	flag.Int64Var(&cfg.ReportInterval, "r", cfg.ReportInterval, "report interval in seconds")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse environment variables: %v", err)
	}

	return cfg
}
