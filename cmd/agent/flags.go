package main

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
	myLog "github.com/squaredbusinessman/go-musthave-metrics/internal/logger"
	"go.uber.org/zap"
)

type Config struct {
	Addr           string
	PollInterval   int
	ReportInterval int
	ReportFormat   string
}

func parseConfig() Config {
	cfg := Config{ReportFormat: agent.ReportFormatPlain}

	flag.StringVar(&cfg.Addr, "a", ":8080", "Run server address")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Poll interval(seconds)")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Report interval(seconds)")
	flag.StringVar(&cfg.ReportFormat, "f", agent.ReportFormatPlain, "Report format: plain or json")

	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Addr = envAddr
	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		if v, err := strconv.Atoi(envPollInterval); err == nil {
			cfg.PollInterval = v
		} else {
			myLog.Log.Warn("ignoring POLL_INTERVAL=",
				zap.String("value", envPollInterval),
				zap.Error(err),
			)
		}
	}

	if envRepInterval := os.Getenv("REPORT_INTERVAL"); envRepInterval != "" {
		if v, err := strconv.Atoi(envRepInterval); err == nil {
			cfg.ReportInterval = v
		} else {
			myLog.Log.Warn("ignoring REPORT_INTERVAL=",
				zap.String("value", envRepInterval),
				zap.Error(err),
			)
		}
	}

	if envFormat := os.Getenv("REPORT_FORMAT"); envFormat != "" {
		cfg.ReportFormat = envFormat
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
