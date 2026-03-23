package main

import "testing"

func TestParseConfigArgsAuditFromFlags(t *testing.T) {
	cfg, err := parseConfigArgs([]string{"--audit-file", "/tmp/audit.log", "--audit-url", "http://localhost:8081/audit"})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Audit.FilePath, "/tmp/audit.log"; got != want {
		t.Fatalf("Audit.FilePath = %q, want %q", got, want)
	}
	if got, want := cfg.Audit.URL, "http://localhost:8081/audit"; got != want {
		t.Fatalf("Audit.URL = %q, want %q", got, want)
	}
}

func TestParseConfigArgsAuditFromEnv(t *testing.T) {
	t.Setenv("AUDIT_FILE", "/tmp/env-audit.log")
	t.Setenv("AUDIT_URL", "http://localhost:8082/audit")

	cfg, err := parseConfigArgs(nil)
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Audit.FilePath, "/tmp/env-audit.log"; got != want {
		t.Fatalf("Audit.FilePath = %q, want %q", got, want)
	}
	if got, want := cfg.Audit.URL, "http://localhost:8082/audit"; got != want {
		t.Fatalf("Audit.URL = %q, want %q", got, want)
	}
}
