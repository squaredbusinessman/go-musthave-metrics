package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	appconfig "github.com/squaredbusinessman/go-musthave-metrics/internal/config"
)

// Config - конфигурация агента отправки метрик.
type Config struct {
	Addr           string `env:"ADDRESS"`
	GRPCAddr       string `env:"GRPC_ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	ReportFormat   string `env:"REPORT_FORMAT"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

type fileConfig struct {
	Address        *string                    `json:"address"`
	GRPCAddress    *string                    `json:"grpc_address"`
	PollInterval   *appconfig.DurationSeconds `json:"poll_interval"`
	ReportInterval *appconfig.DurationSeconds `json:"report_interval"`
	ReportFormat   *string                    `json:"report_format"`
	Key            *string                    `json:"key"`
	RateLimit      *int                       `json:"rate_limit"`
	CryptoKey      *string                    `json:"crypto_key"`
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
		Addr:           ":8080",
		PollInterval:   2,
		ReportInterval: 10,
		ReportFormat:   agent.ReportFormatPlain,
		RateLimit:      1,
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

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.Addr, "a", cfg.Addr, "Run server address")
	fs.StringVar(&cfg.GRPCAddr, "g", cfg.GRPCAddr, "gRPC server address")
	fs.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval(seconds)")
	fs.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval(seconds)")
	fs.StringVar(&cfg.ReportFormat, "f", cfg.ReportFormat, "Report format: plain or json")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	fs.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limiting")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Crypto key")
	fs.StringVar(&configPath, "c", configPath, "config file path")
	fs.StringVar(&configPath, "config", configPath, "config file path")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Printf("(agent) ignoring env vars due to error: %v", err)
	}

	cfg.ReportFormat = normalizeReportFormat(cfg.ReportFormat)

	return cfg, nil
}

func normalizeReportFormat(value string) string {
	switch strings.ToLower(value) {
	case agent.ReportFormatJSON:
		return agent.ReportFormatJSON
	default:
		return agent.ReportFormatPlain
	}
}

func (fc fileConfig) apply(cfg *Config) {
	if fc.Address != nil {
		cfg.Addr = *fc.Address
	}
	if fc.GRPCAddress != nil {
		cfg.GRPCAddr = *fc.GRPCAddress
	}
	if fc.PollInterval != nil {
		cfg.PollInterval = fc.PollInterval.Seconds()
	}
	if fc.ReportInterval != nil {
		cfg.ReportInterval = fc.ReportInterval.Seconds()
	}
	if fc.ReportFormat != nil {
		cfg.ReportFormat = *fc.ReportFormat
	}
	if fc.Key != nil {
		cfg.Key = *fc.Key
	}
	if fc.RateLimit != nil {
		cfg.RateLimit = *fc.RateLimit
	}
	if fc.CryptoKey != nil {
		cfg.CryptoKey = *fc.CryptoKey
	}
}
