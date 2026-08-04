package server

import "flag"

type Config struct {
	ServerAddr string
}

func GetConfig() Config {
	cfg := Config{
		ServerAddr: "localhost:8080",
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address (host:port)")
	flag.Parse()

	return cfg
}
