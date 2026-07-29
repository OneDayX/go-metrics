package agent

import (
	"flag"
)

type Config struct {
	PollInterval   int64
	ReportInterval int64
	ServerAddr     string
}

func DefaultConfig() Config {
	return Config{
		PollInterval:   2,
		ReportInterval: 10,
		ServerAddr:     "localhost:8080",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server address (host:port)")
	flag.Int64Var(&c.PollInterval, "p", c.PollInterval, "poll interval in seconds")
	flag.Int64Var(&c.ReportInterval, "r", c.ReportInterval, "report interval in seconds")
	flag.Parse()
}
