package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

type Config struct {
	RunAddr            string
	LogLevel           string
	StoreInterval      int
	FileStorageEnabled bool
	FileStoragePath    string
	Restore            bool
	DatabaseDSN        string
}

func parseConfig() Config {
	cfg := Config{
		StoreInterval:      300,
		FileStorageEnabled: false,
		FileStoragePath:    "/tmp/devops-metrics-db.json",
		Restore:            true,
	}

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "Run server address")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "store interval in seconds (0 for sync)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore metrics from file on startup")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN")

	flag.Parse()

	// Не понимаю, насколько это кринж.
	// Суть в том чтобы чекнуть, был ли флаг -f передан явно.
	fileFlagSet := false

	flag.CommandLine.Visit(
		func(f *flag.Flag) {
			if f.Name == "f" {
				fileFlagSet = true
			}
		})

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

	// тут включаем файл только если явно задан env или флаг -f
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
		cfg.FileStorageEnabled = true
	} else if fileFlagSet && cfg.FileStoragePath != "" {
		cfg.FileStorageEnabled = true
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if v, err := strconv.ParseBool(envRestore); err == nil {
			cfg.Restore = v
		} else {
			log.Printf("ignoring RESTORE=%q: %v", envRestore, err)
		}
	}

	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		cfg.DatabaseDSN = envDSN
	}

	if cfg.StoreInterval < 0 {
		cfg.StoreInterval = 0
	}

	return cfg
}
