package main

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	Addr           string
	PollInterval   int
	ReportInterval int
}

func parseConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Addr, "a", ":8080", "Run server address")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Poll interval(seconds)")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Report interval(seconds)")

	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Addr = envAddr
	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		if v, err := strconv.Atoi(envPollInterval); err == nil {
			cfg.PollInterval = v
		}
	}

	if envRepInterval := os.Getenv("REPORT_INTERVAL"); envRepInterval != "" {
		if v, err := strconv.Atoi(envRepInterval); err == nil {
			cfg.ReportInterval = v
		}
	}

	return cfg
}
