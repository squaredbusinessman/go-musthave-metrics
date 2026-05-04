package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	appconfig "github.com/squaredbusinessman/go-musthave-metrics/internal/config"
)

// Config - итоговая конфигурация сервера.
type Config struct {
	Server   ServerConfig
	Storage  StorageConfig
	Database DBConfig
	Audit    AuditConfig
	Crypto   CryptoConfig
}

// DBConfig - настройки подключения к PostgreSQL.
type DBConfig struct {
	DSN string `env:"DATABASE_DSN"`
}

// ServerConfig - сетевые и общие настройки HTTP-сервера.
type ServerConfig struct {
	RunAddr       string `env:"ADDRESS"`
	LogLevel      string `env:"LOG_LEVEL"`
	Key           string `env:"KEY"`
	TrustedSubnet string `env:"TRUSTED_SUBNET"`
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

// CryptoConfig - настройки шифрования
type CryptoConfig struct {
	KeyPath string `env:"CRYPTO_KEY"`
}

type fileConfig struct {
	Address       *string                    `json:"address"`
	LogLevel      *string                    `json:"log_level"`
	Key           *string                    `json:"key"`
	Restore       *bool                      `json:"restore"`
	StoreInterval *appconfig.DurationSeconds `json:"store_interval"`
	StoreFile     *string                    `json:"store_file"`
	DatabaseDSN   *string                    `json:"database_dsn"`
	AuditFile     *string                    `json:"audit_file"`
	AuditURL      *string                    `json:"audit_url"`
	CryptoKey     *string                    `json:"crypto_key"`
	TrustedSubnet *string                    `json:"trusted_subnet"`
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

	configPath, err := appconfig.DiscoverPath(args, "CONFIG")
	if err != nil {
		return Config{}, err
	}

	var loadedFileConfig fileConfig
	if configPath != "" {
		if err := appconfig.ReadJSONFile(configPath, &loadedFileConfig); err != nil {
			return Config{}, err
		}
		loadedFileConfig.apply(&cfg)
	}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.Server.RunAddr, "a", cfg.Server.RunAddr, "Run server address")
	fs.StringVar(&cfg.Server.LogLevel, "l", cfg.Server.LogLevel, "log level")
	fs.StringVar(&cfg.Server.Key, "k", cfg.Server.Key, "Hash key")
	fs.StringVar(&cfg.Server.TrustedSubnet, "t", cfg.Server.TrustedSubnet, "trusted subnet in CIDR notation")
	fs.IntVar(&cfg.Storage.StoreInterval, "i", cfg.Storage.StoreInterval, "store interval in seconds (0 for sync)")
	fs.StringVar(&cfg.Storage.FileStoragePath, "f", cfg.Storage.FileStoragePath, "file storage path")
	fs.BoolVar(&cfg.Storage.Restore, "r", cfg.Storage.Restore, "restore metrics from file on startup")
	fs.StringVar(&cfg.Database.DSN, "d", cfg.Database.DSN, "database DSN")
	fs.StringVar(&cfg.Audit.FilePath, "audit-file", cfg.Audit.FilePath, "audit log file path")
	fs.StringVar(&cfg.Audit.URL, "audit-url", cfg.Audit.URL, "audit receiver URL")
	fs.StringVar(&cfg.Crypto.KeyPath, "crypto-key", cfg.Crypto.KeyPath, "crypto key")
	fs.StringVar(&configPath, "c", configPath, "config file path")
	fs.StringVar(&configPath, "config", configPath, "config file path")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

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

	_, envFilePathSet := os.LookupEnv("FILE_STORAGE_PATH")

	// тут включаем файл только если путь задан явно через config, env или флаг -f
	switch {
	case envFilePathSet:
		cfg.Storage.FileStorageEnabled = cfg.Storage.FileStoragePath != ""
	case fileFlagSet:
		cfg.Storage.FileStorageEnabled = cfg.Storage.FileStoragePath != ""
	case loadedFileConfig.StoreFile != nil:
		cfg.Storage.FileStorageEnabled = cfg.Storage.FileStoragePath != ""
	default:
		cfg.Storage.FileStorageEnabled = false
	}

	if cfg.Storage.StoreInterval < 0 {
		cfg.Storage.StoreInterval = 0
	}

	if cfg.Server.TrustedSubnet != "" {
		if _, _, err := net.ParseCIDR(cfg.Server.TrustedSubnet); err != nil {
			return Config{}, err
		}
	}

	return cfg, nil
}

func (fc fileConfig) apply(cfg *Config) {
	if fc.Address != nil {
		cfg.Server.RunAddr = *fc.Address
	}
	if fc.LogLevel != nil {
		cfg.Server.LogLevel = *fc.LogLevel
	}
	if fc.Key != nil {
		cfg.Server.Key = *fc.Key
	}
	if fc.Restore != nil {
		cfg.Storage.Restore = *fc.Restore
	}
	if fc.StoreInterval != nil {
		cfg.Storage.StoreInterval = fc.StoreInterval.Seconds()
	}
	if fc.StoreFile != nil {
		cfg.Storage.FileStoragePath = *fc.StoreFile
	}
	if fc.DatabaseDSN != nil {
		cfg.Database.DSN = *fc.DatabaseDSN
	}
	if fc.AuditFile != nil {
		cfg.Audit.FilePath = *fc.AuditFile
	}
	if fc.AuditURL != nil {
		cfg.Audit.URL = *fc.AuditURL
	}
	if fc.CryptoKey != nil {
		cfg.Crypto.KeyPath = *fc.CryptoKey
	}
	if fc.TrustedSubnet != nil {
		cfg.Server.TrustedSubnet = *fc.TrustedSubnet
	}
}
