package main

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestParseConfigArgsCryptoKeyFromFlag(t *testing.T) {
	cfg, err := parseConfigArgs([]string{"-crypto-key", "/tmp/private.pem"})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Crypto.KeyPath, "/tmp/private.pem"; got != want {
		t.Fatalf("Crypto.KeyPath = %q, want %q", got, want)
	}
}

func TestParseConfigArgsCryptoKeyFromEnv(t *testing.T) {
	t.Setenv("CRYPTO_KEY", "/tmp/env-private.pem")

	cfg, err := parseConfigArgs(nil)
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Crypto.KeyPath, "/tmp/env-private.pem"; got != want {
		t.Fatalf("Crypto.KeyPath = %q, want %q", got, want)
	}
}

func TestParseConfigArgsTrustedSubnetFromFlag(t *testing.T) {
	cfg, err := parseConfigArgs([]string{"-t", "192.168.1.0/24"})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Server.TrustedSubnet, "192.168.1.0/24"; got != want {
		t.Fatalf("Server.TrustedSubnet = %q, want %q", got, want)
	}
}

func TestParseConfigArgsTrustedSubnetFromEnv(t *testing.T) {
	t.Setenv("TRUSTED_SUBNET", "10.10.0.0/16")

	cfg, err := parseConfigArgs(nil)
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Server.TrustedSubnet, "10.10.0.0/16"; got != want {
		t.Fatalf("Server.TrustedSubnet = %q, want %q", got, want)
	}
}

func TestParseConfigArgsRejectsInvalidTrustedSubnet(t *testing.T) {
	if _, err := parseConfigArgs([]string{"-t", "not-a-cidr"}); err == nil {
		t.Fatalf("expected invalid trusted subnet error")
	}
}

func TestParseConfigArgsFromJSONConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "server-config.json")
	configData := []byte(`{
		"address": "localhost:9090",
		"log_level": "debug",
		"key": "secret",
		"restore": false,
		"store_interval": "5s",
		"store_file": "/tmp/server.json",
		"database_dsn": "postgres://localhost/db",
		"audit_file": "/tmp/audit.log",
		"audit_url": "http://localhost:8081/audit",
		"crypto_key": "/tmp/private.pem",
		"trusted_subnet": "172.16.0.0/12"
	}`)
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := parseConfigArgs([]string{"-c", configPath})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Server.RunAddr, "localhost:9090"; got != want {
		t.Fatalf("Server.RunAddr = %q, want %q", got, want)
	}
	if got, want := cfg.Server.LogLevel, "debug"; got != want {
		t.Fatalf("Server.LogLevel = %q, want %q", got, want)
	}
	if got, want := cfg.Server.Key, "secret"; got != want {
		t.Fatalf("Server.Key = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Restore, false; got != want {
		t.Fatalf("Storage.Restore = %v, want %v", got, want)
	}
	if got, want := cfg.Storage.StoreInterval, 5; got != want {
		t.Fatalf("Storage.StoreInterval = %d, want %d", got, want)
	}
	if got, want := cfg.Storage.FileStoragePath, "/tmp/server.json"; got != want {
		t.Fatalf("Storage.FileStoragePath = %q, want %q", got, want)
	}
	if !cfg.Storage.FileStorageEnabled {
		t.Fatalf("Storage.FileStorageEnabled = false, want true")
	}
	if got, want := cfg.Database.DSN, "postgres://localhost/db"; got != want {
		t.Fatalf("Database.DSN = %q, want %q", got, want)
	}
	if got, want := cfg.Audit.FilePath, "/tmp/audit.log"; got != want {
		t.Fatalf("Audit.FilePath = %q, want %q", got, want)
	}
	if got, want := cfg.Audit.URL, "http://localhost:8081/audit"; got != want {
		t.Fatalf("Audit.URL = %q, want %q", got, want)
	}
	if got, want := cfg.Crypto.KeyPath, "/tmp/private.pem"; got != want {
		t.Fatalf("Crypto.KeyPath = %q, want %q", got, want)
	}
	if got, want := cfg.Server.TrustedSubnet, "172.16.0.0/12"; got != want {
		t.Fatalf("Server.TrustedSubnet = %q, want %q", got, want)
	}
}

func TestParseConfigArgsPriorityOverJSONConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "server-config.json")
	configData := []byte(`{
		"address": "localhost:9090",
		"restore": false,
		"store_interval": "5s",
		"store_file": "/tmp/from-config.json",
		"database_dsn": "postgres://config/db",
		"crypto_key": "/tmp/config.pem",
		"trusted_subnet": "172.16.0.0/12"
	}`)
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("DATABASE_DSN", "postgres://env/db")
	t.Setenv("TRUSTED_SUBNET", "10.0.0.0/8")

	cfg, err := parseConfigArgs([]string{
		"-c", configPath,
		"-r=true",
		"-i", "9",
		"-f", "/tmp/from-flag.json",
		"-crypto-key", "/tmp/flag.pem",
	})
	if err != nil {
		t.Fatalf("parseConfigArgs() error = %v", err)
	}

	if got, want := cfg.Server.RunAddr, "localhost:7070"; got != want {
		t.Fatalf("Server.RunAddr = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Restore, true; got != want {
		t.Fatalf("Storage.Restore = %v, want %v", got, want)
	}
	if got, want := cfg.Storage.StoreInterval, 9; got != want {
		t.Fatalf("Storage.StoreInterval = %d, want %d", got, want)
	}
	if got, want := cfg.Storage.FileStoragePath, "/tmp/from-flag.json"; got != want {
		t.Fatalf("Storage.FileStoragePath = %q, want %q", got, want)
	}
	if !cfg.Storage.FileStorageEnabled {
		t.Fatalf("Storage.FileStorageEnabled = false, want true")
	}
	if got, want := cfg.Database.DSN, "postgres://env/db"; got != want {
		t.Fatalf("Database.DSN = %q, want %q", got, want)
	}
	if got, want := cfg.Crypto.KeyPath, "/tmp/flag.pem"; got != want {
		t.Fatalf("Crypto.KeyPath = %q, want %q", got, want)
	}
	if got, want := cfg.Server.TrustedSubnet, "10.0.0.0/8"; got != want {
		t.Fatalf("Server.TrustedSubnet = %q, want %q", got, want)
	}
}
