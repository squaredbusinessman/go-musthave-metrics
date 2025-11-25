package main

import (
	"flag"
	"time"
)

type Config struct {
	Addr           string
	PollInterval   int
	ReportInterval int
}

func parseConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Addr, "a", ":8080", "Run server address")
	flag.DurationVar(&cfg.PollInterval, "p", 2, "Poll interval(seconds)")
	flag.DurationVar(&cfg.ReportInterval, "r", 10, "Report interval(seconds)")

	flag.Parse()

	return cfg
}
