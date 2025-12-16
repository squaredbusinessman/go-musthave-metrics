package main

import (
	"flag"
	"os"
)

type Config struct {
	RunAddr string
}

func parseConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "Run server address")

	flag.Parse()

	if envRunAddr := os.Getenv("RUN_ADDR"); envRunAddr != "" {
		cfg.RunAddr = envRunAddr
	}

	return cfg
}
