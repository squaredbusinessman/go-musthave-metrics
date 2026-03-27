package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config - итоговая конфигурация сервера.
type Config struct {
	Server   ServerConfig
	Storage  StorageConfig
	Database DBConfig
	Audit    AuditConfig
}

// DBConfig - настройки подключения к PostgreSQL.
type DBConfig struct {
	DSN string `env:"DATABASE_DSN"`
}

// ServerConfig - сетевые и общие настройки HTTP-сервера.
type ServerConfig struct {
	RunAddr  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
	Key      string `env:"KEY"`
}

// StorageConfig - настройки файлового хранилища метрик.
type StorageConfig struct {
	StoreInterval      int `env:"STORE_INTERVAL"`
	FileStorageEnabled bool
	FileStoragePath    string `env:"FILE_STORAGE_PATH"`
	Restore            bool   `env:"RESTORE"`
}

// AuditConfig - настройки приемников аудита.
type AuditConfig struct {
	FilePath string `env:"AUDIT_FILE"`
	URL      string `env:"AUDIT_URL"`
}

func parseConfig() Config {
	cfg, err := parseConfigArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("parse config failure: %v", err)
	}
	return cfg
}

func parseConfigArgs(args []string) (Config, error) {
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

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.Server.RunAddr, "a", cfg.Server.RunAddr, "Run server address")
	fs.StringVar(&cfg.Server.LogLevel, "l", cfg.Server.LogLevel, "log level")
	fs.StringVar(&cfg.Server.Key, "k", "", "Hash key")
	fs.IntVar(&cfg.Storage.StoreInterval, "i", cfg.Storage.StoreInterval, "store interval in seconds (0 for sync)")
	fs.StringVar(&cfg.Storage.FileStoragePath, "f", cfg.Storage.FileStoragePath, "file storage path")
	fs.BoolVar(&cfg.Storage.Restore, "r", cfg.Storage.Restore, "restore metrics from file on startup")
	fs.StringVar(&cfg.Database.DSN, "d", "", "database DSN")
	fs.StringVar(&cfg.Audit.FilePath, "audit-file", "", "audit log file path")
	fs.StringVar(&cfg.Audit.URL, "audit-url", "", "audit receiver URL")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// Не понимаю, насколько это кринж.
	// Суть в том чтобы чекнуть, был ли флаг -f передан явно.
	fileFlagSet := false

	fs.Visit(
		func(f *flag.Flag) {
			if f.Name == "f" {
				fileFlagSet = true
			}
		})

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Printf("(server) ignoring env vars due to error: %v", err)
	}

	// Для аудита отдельный флаг не нужен.
	// Если путь к файлу или URL заданы, приёмник считается включённым.

	// тут включаем файл только если явно задан env или флаг -f
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.Storage.FileStorageEnabled = true
	} else if fileFlagSet && cfg.Storage.FileStoragePath != "" {
		cfg.Storage.FileStorageEnabled = true
	}

	if cfg.Storage.StoreInterval < 0 {
		cfg.Storage.StoreInterval = 0
	}

	return cfg, nil
}
