package main

import (
	"flag"
	"time"
)

type Config struct {
	Addr           string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func parseConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Addr, "a", ":8080", "Run server address")
	flag.DurationVar(&cfg.PollInterval, "p", 2*time.Second, "Poll interval")
	flag.DurationVar(&cfg.ReportInterval, "r", 10*time.Second, "Report interval")

	flag.Parse()

	return cfg
}
