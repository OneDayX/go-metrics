package agent

import "time"

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		ServerAddr:     "localhost:8080",
	}
}
