package agent

import (
	"flag"
	"time"
)

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
}

func DefaultConfig() Config {
	return Config{
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		ServerAddr:     "localhost:8080",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server address (host:port)")
	flag.DurationVar(&c.PollInterval, "p", c.PollInterval, "poll interval (e.g. 2s, 500ms)")
	flag.DurationVar(&c.ReportInterval, "r", c.ReportInterval, "report interval (e.g. 10s, 1m)")
	flag.Parse()
}
