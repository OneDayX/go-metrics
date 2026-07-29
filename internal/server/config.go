package server

import "flag"

type Config struct {
	ServerAddr string
}

func DefaultConfig() Config {
	return Config{
		ServerAddr: "localhost:8080",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server address (host:port)")
	flag.Parse()
}
