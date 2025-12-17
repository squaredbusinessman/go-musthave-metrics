package main

import (
	"flag"
	"os"
)

type Config struct {
	RunAddr  string
	LogLevel string
}

func parseConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "Run server address")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")

	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.RunAddr = envAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	return cfg
}
