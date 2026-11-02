package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_ACCESS_SECRET", "01234567890123456789012345678901")
}

func TestLoadDefaultsAndTrustedProxies(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.0/8,10.0.0.0/8")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Address != ":8080" || cfg.HTTP.MaxBodyBytes != 1<<20 || cfg.HTTP.MaxHeaderBytes != 32<<10 {
		t.Fatalf("unexpected HTTP defaults: %#v", cfg.HTTP)
	}
	if cfg.Places.PublicCacheTTL != 60*time.Second || cfg.Places.PublicRateLimit != 60 || cfg.Places.AdminRateLimit != 30 {
		t.Fatalf("unexpected Places defaults: %#v", cfg.Places)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Fatalf("trusted proxies = %d", len(cfg.TrustedProxies))
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	cases := map[string]func(*testing.T){
		"missing database":       func(t *testing.T) { setRequiredEnv(t); t.Setenv("DATABASE_URL", "") },
		"short JWT secret":       func(t *testing.T) { setRequiredEnv(t); t.Setenv("JWT_ACCESS_SECRET", "short") },
		"invalid environment":    func(t *testing.T) { setRequiredEnv(t); t.Setenv("APP_ENV", "staging") },
		"invalid duration":       func(t *testing.T) { setRequiredEnv(t); t.Setenv("HTTP_READ_TIMEOUT", "later") },
		"invalid CIDR":           func(t *testing.T) { setRequiredEnv(t); t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr") },
		"min exceeds max conns":  func(t *testing.T) { setRequiredEnv(t); t.Setenv("DB_MIN_CONNS", "11"); t.Setenv("DB_MAX_CONNS", "10") },
		"header limit too small": func(t *testing.T) { setRequiredEnv(t); t.Setenv("HTTP_MAX_HEADER_BYTES", "4096") },
		"header limit too large": func(t *testing.T) { setRequiredEnv(t); t.Setenv("HTTP_MAX_HEADER_BYTES", "65536") },
		"sub-second rate window": func(t *testing.T) { setRequiredEnv(t); t.Setenv("RATE_LIMIT_WINDOW_SECONDS", "0") },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			setup(t)
			if _, err := Load(); err == nil || strings.TrimSpace(err.Error()) == "" {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}
