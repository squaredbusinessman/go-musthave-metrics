package main

import (
	"flag"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

// Config - конфигурация агента отправки метрик.
type Config struct {
	Addr           string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	ReportFormat   string `env:"REPORT_FORMAT"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

func parseConfig() Config {
	cfg := Config{ReportFormat: agent.ReportFormatPlain}

	flag.StringVar(&cfg.Addr, "a", ":8080", "Run server address")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Poll interval(seconds)")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Report interval(seconds)")
	flag.StringVar(&cfg.ReportFormat, "f", agent.ReportFormatPlain, "Report format: plain or json")
	flag.StringVar(&cfg.Key, "k", "", "Hash key")
	flag.IntVar(&cfg.RateLimit, "l", 1, "Rate limiting")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "Crypto key")

	flag.Parse()

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		myLog.Log.Warn("(agent) ignoring env vars due to error", zap.Error(err))
	}

	cfg.ReportFormat = normalizeReportFormat(cfg.ReportFormat)

	return cfg
}

func normalizeReportFormat(value string) string {
	switch strings.ToLower(value) {
	case agent.ReportFormatJSON:
		return agent.ReportFormatJSON
	default:
		return agent.ReportFormatPlain
	}
}
