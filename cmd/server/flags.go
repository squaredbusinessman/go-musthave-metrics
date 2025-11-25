package main

import "flag"

var flagRunAddr string

type Config struct {
	RunAddr string
}

func parseConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "Run server address")

	flag.Parse()

	return cfg
}
