package config_test

import (
	"testing"

	"github.com/ZacharyJo/mysql-cli-go/internal/config"
)

func TestBuildMySQLDSN(t *testing.T) {
	cfg := &config.Config{
		User:           "root",
		Password:       "secret",
		Database:       "mydb",
		ConnectTimeout: 10,
		SSLMode:        "disable",
	}
	dsn := config.BuildMySQLDSN(cfg, "127.0.0.1:3306")
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
	if !contains(dsn, "root:secret@tcp(127.0.0.1:3306)/mydb") {
		t.Errorf("DSN missing expected parts, got: %s", dsn)
	}
}

func TestBuildMySQLDSN_WithTLS(t *testing.T) {
	cfg := &config.Config{
		User:           "admin",
		Password:       "pass",
		Database:       "db",
		ConnectTimeout: 5,
		SSLMode:        "verify-full",
	}
	dsn := config.BuildMySQLDSN(cfg, "10.0.0.1:3306")
	if !contains(dsn, "tls=custom") {
		t.Errorf("expected tls=custom in DSN, got: %s", dsn)
	}
}

func TestBuildPgxDSN(t *testing.T) {
	cfg := &config.Config{
		User:           "gaussadmin",
		Password:       "pass",
		Database:       "gaussdb",
		ConnectTimeout: 10,
		SSLMode:        "disable",
	}
	dsn := config.BuildPgxDSN(cfg, "10.0.0.5", "5432")
	if !contains(dsn, "host=10.0.0.5") {
		t.Errorf("DSN missing host, got: %s", dsn)
	}
	if !contains(dsn, "port=5432") {
		t.Errorf("DSN missing port, got: %s", dsn)
	}
	if !contains(dsn, "sslmode=disable") {
		t.Errorf("DSN missing sslmode, got: %s", dsn)
	}
}

func TestBuildPgxDSN_WithSSLCA(t *testing.T) {
	cfg := &config.Config{
		User:           "user",
		Password:       "pass",
		Database:       "db",
		ConnectTimeout: 10,
		SSLMode:        "verify-ca",
		SSLCA:          "/etc/ssl/ca.pem",
	}
	dsn := config.BuildPgxDSN(cfg, "localhost", "5432")
	if !contains(dsn, "sslrootcert=/etc/ssl/ca.pem") {
		t.Errorf("DSN missing sslrootcert, got: %s", dsn)
	}
}

func TestEffectiveHost_Master(t *testing.T) {
	cfg := &config.Config{Master: "10.0.0.1:3306", Host: "localhost", Port: 3306}
	if got := cfg.EffectiveHost(); got != "10.0.0.1:3306" {
		t.Errorf("expected master address, got %s", got)
	}
}

func TestEffectiveHost_HostPort(t *testing.T) {
	cfg := &config.Config{Host: "127.0.0.1", Port: 5432}
	if got := cfg.EffectiveHost(); got != "127.0.0.1:5432" {
		t.Errorf("expected host:port, got %s", got)
	}
}

func TestIsMySQLCompatible(t *testing.T) {
	for _, dbType := range []string{"mysql", "oceanbase"} {
		cfg := &config.Config{DBType: dbType}
		if !cfg.IsMySQLCompatible() {
			t.Errorf("%s should be MySQL-compatible", dbType)
		}
	}
}

func TestIsPgCompatible(t *testing.T) {
	for _, dbType := range []string{"gaussdb", "kingbase"} {
		cfg := &config.Config{DBType: dbType}
		if !cfg.IsPgCompatible() {
			t.Errorf("%s should be PG-compatible", dbType)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
