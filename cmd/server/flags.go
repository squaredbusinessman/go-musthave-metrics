package main

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	RunAddr            string `env:"ADDRESS"`
	LogLevel           string `env:"LOG_LEVEL"`
	StoreInterval      int    `env:"STORE_INTERVAL"`
	FileStorageEnabled bool
	FileStoragePath    string `env:"FILE_STORAGE_PATH"`
	Restore            bool   `env:"RESTORE"`
	DatabaseDSN        string `env:"DATABASE_DSN"`
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

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Printf("(server) ignoring env vars due to error: %v", err)
	}

	// тут включаем файл только если явно задан env или флаг -f
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStorageEnabled = true
	} else if fileFlagSet && cfg.FileStoragePath != "" {
		cfg.FileStorageEnabled = true
	}

	if cfg.StoreInterval < 0 {
		cfg.StoreInterval = 0
	}

	return cfg
}
