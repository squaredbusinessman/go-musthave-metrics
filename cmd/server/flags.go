package main

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server   ServerConfig
	Storage  StorageConfig
	Database DBConfig
}

type DBConfig struct {
	DSN string `env:"DATABASE_DSN"`
}

type ServerConfig struct {
	RunAddr  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
	Key      string `env:"KEY"`
}

type StorageConfig struct {
	StoreInterval      int `env:"STORE_INTERVAL"`
	FileStorageEnabled bool
	FileStoragePath    string `env:"FILE_STORAGE_PATH"`
	Restore            bool   `env:"RESTORE"`
}

func parseConfig() Config {
	cfg := Config{
		Server: ServerConfig{
			RunAddr:  ":8080",
			LogLevel: "info",
		},
		Storage: StorageConfig{
			StoreInterval:      300,
			FileStorageEnabled: false,
			FileStoragePath:    "/tmp/devops-metrics-db.json",
			Restore:            true,
		},
	}

	flag.StringVar(&cfg.Server.RunAddr, "a", cfg.Server.RunAddr, "Run server address")
	flag.StringVar(&cfg.Server.LogLevel, "l", cfg.Server.LogLevel, "log level")
	flag.StringVar(&cfg.Server.Key, "k", "", "Hash key")
	flag.IntVar(&cfg.Storage.StoreInterval, "i", cfg.Storage.StoreInterval, "store interval in seconds (0 for sync)")
	flag.StringVar(&cfg.Storage.FileStoragePath, "f", cfg.Storage.FileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Storage.Restore, "r", cfg.Storage.Restore, "restore metrics from file on startup")
	flag.StringVar(&cfg.Database.DSN, "d", "", "database DSN")

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
		cfg.Storage.FileStorageEnabled = true
	} else if fileFlagSet && cfg.Storage.FileStoragePath != "" {
		cfg.Storage.FileStorageEnabled = true
	}

	if cfg.Storage.StoreInterval < 0 {
		cfg.Storage.StoreInterval = 0
	}

	return cfg
}
