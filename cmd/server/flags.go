package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

type Config struct {
	RunAddr         string
	LogLevel        string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

func parseConfig() Config {
	cfg := Config{
		StoreInterval:   300,
		FileStoragePath: "/tmp/devops-metrics-db.json",
		Restore:         true,
	}

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "Run server address")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "store interval in seconds (0 for sync)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file on startup")

	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.RunAddr = envAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		if v, err := strconv.Atoi(envInterval); err == nil {
			cfg.StoreInterval = v
		} else {
			log.Printf("ignoring STORE_INTERVAL=%q: %v", envInterval, err)
		}
	}

	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if v, err := strconv.ParseBool(envRestore); err == nil {
			cfg.Restore = v
		} else {
			log.Printf("ignoring RESTORE=%q: %v", envRestore, err)
		}
	}

	if cfg.StoreInterval < 0 {
		cfg.StoreInterval = 0
	}

	return cfg
}
