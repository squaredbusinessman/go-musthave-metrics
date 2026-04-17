package main

import (
	"flag"
	"io"
	"os"
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

func TestParseConfig(t *testing.T) {
	oldCommandLine := flag.CommandLine
	oldArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = oldCommandLine
		os.Args = oldArgs
	})

	flag.CommandLine = flag.NewFlagSet("agent", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = []string{
		"agent",
		"-a",
		"localhost:9090",
		"-p",
		"3",
		"-r",
		"12",
		"-f",
		"JSON",
		"-k",
		"secret",
		"-l",
		"4",
		"-crypto-key",
		"/tmp/public.pem",
	}

	cfg := parseConfig()

	if cfg.Addr != "localhost:9090" {
		t.Fatalf("Addr = %q, want %q", cfg.Addr, "localhost:9090")
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
