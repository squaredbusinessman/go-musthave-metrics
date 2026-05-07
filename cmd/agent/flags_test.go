package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/squaredbusinessman/go-musthave-metrics/internal/agent"
)

func TestNormalizeReportFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "json", want: agent.ReportFormatJSON},
		{input: "JSON", want: agent.ReportFormatJSON},
		{input: "plain", want: agent.ReportFormatPlain},
		{input: "unexpected", want: agent.ReportFormatPlain},
	}

	for _, tt := range tests {
		if got := normalizeReportFormat(tt.input); got != tt.want {
			t.Fatalf("normalizeReportFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseConfigArgs(t *testing.T) {
	cfg, err := parseConfigArgs([]string{
		"-a", "localhost:9090",
		"-g", "localhost:3200",
		"-p", "3",
		"-r", "12",
		"-f", "JSON",
		"-k", "secret",
		"-l", "4",
		"-crypto-key", "/tmp/public.pem",
	})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if cfg.Addr != "localhost:9090" {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, "localhost:9090")
	}
	if cfg.GRPCAddr != "localhost:3200" {
		t.Fatalf("GRPCAddr = %q, want %q", cfg.GRPCAddr, "localhost:3200")
	}
	if cfg.PollInterval != 3 {
		t.Fatalf("PollInterval = %d, want 3", cfg.PollInterval)
	}
	if cfg.ReportInterval != 12 {
		t.Fatalf("ReportInterval = %d, want 12", cfg.ReportInterval)
	}
	if cfg.ReportFormat != agent.ReportFormatJSON {
		t.Fatalf("ReportFormat = %q, want %q", cfg.ReportFormat, agent.ReportFormatJSON)
	}
	if cfg.Key != "secret" {
		t.Fatalf("Key = %q, want %q", cfg.Key, "secret")
	}
	if cfg.RateLimit != 4 {
		t.Fatalf("RateLimit = %d, want 4", cfg.RateLimit)
	}
	if cfg.CryptoKey != "/tmp/public.pem" {
		t.Fatalf("CryptoKey = %q, want %q", cfg.CryptoKey, "/tmp/public.pem")
	}
}

func TestParseConfigArgsFromJSONConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "agent-config.json")
	configData := []byte(`{
		"address": "localhost:9090",
		"grpc_address": "localhost:3200",
		"poll_interval": "3s",
		"report_interval": "12s",
		"report_format": "JSON",
		"key": "secret",
		"rate_limit": 4,
		"crypto_key": "/tmp/public.pem"
	}`)
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := parseConfigArgs([]string{"-c", configPath})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Addr, "localhost:9090"; got != want {
		t.Fatalf("Addr = %q, want %q", got, want)
	}
	if got, want := cfg.GRPCAddr, "localhost:3200"; got != want {
		t.Fatalf("GRPCAddr = %q, want %q", got, want)
	}
	if got, want := cfg.PollInterval, 3; got != want {
		t.Fatalf("PollInterval = %d, want %d", got, want)
	}
	if got, want := cfg.ReportInterval, 12; got != want {
		t.Fatalf("ReportInterval = %d, want %d", got, want)
	}
	if got, want := cfg.ReportFormat, agent.ReportFormatJSON; got != want {
		t.Fatalf("ReportFormat = %q, want %q", got, want)
	}
	if got, want := cfg.Key, "secret"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}
	if got, want := cfg.RateLimit, 4; got != want {
		t.Fatalf("RateLimit = %d, want %d", got, want)
	}
	if got, want := cfg.CryptoKey, "/tmp/public.pem"; got != want {
		t.Fatalf("CryptoKey = %q, want %q", got, want)
	}
}

func TestParseConfigArgsPriorityOverJSONConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "agent-config.json")
	configData := []byte(`{
		"address": "localhost:9090",
		"grpc_address": "localhost:3200",
		"poll_interval": "3s",
		"report_interval": "12s",
		"report_format": "plain",
		"key": "config-secret",
		"rate_limit": 4,
		"crypto_key": "/tmp/config-public.pem"
	}`)
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("GRPC_ADDRESS", "localhost:3300")
	t.Setenv("RATE_LIMIT", "8")

	cfg, err := parseConfigArgs([]string{
		"-c", configPath,
		"-p", "6",
		"-r", "18",
		"-f", "json",
		"-k", "flag-secret",
		"-crypto-key", "/tmp/flag-public.pem",
	})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Addr, "localhost:7070"; got != want {
		t.Fatalf("Addr = %q, want %q", got, want)
	}
	if got, want := cfg.GRPCAddr, "localhost:3300"; got != want {
		t.Fatalf("GRPCAddr = %q, want %q", got, want)
	}
	if got, want := cfg.PollInterval, 6; got != want {
		t.Fatalf("PollInterval = %d, want %d", got, want)
	}
	if got, want := cfg.ReportInterval, 18; got != want {
		t.Fatalf("ReportInterval = %d, want %d", got, want)
	}
	if got, want := cfg.ReportFormat, agent.ReportFormatJSON; got != want {
		t.Fatalf("ReportFormat = %q, want %q", got, want)
	}
	if got, want := cfg.Key, "flag-secret"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}
	if got, want := cfg.RateLimit, 8; got != want {
		t.Fatalf("RateLimit = %d, want %d", got, want)
	}
	if got, want := cfg.CryptoKey, "/tmp/flag-public.pem"; got != want {
		t.Fatalf("CryptoKey = %q, want %q", got, want)
	}
}
