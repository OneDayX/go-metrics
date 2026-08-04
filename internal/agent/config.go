package agent

import (
	"flag"
)

type Config struct {
	PollInterval   int64
	ReportInterval int64
	ServerAddr     string
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

	return cfg
}
