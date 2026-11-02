package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment    string
	HTTP           HTTP
	Database       Database
	Redis          Redis
	JWT            JWT
	Places         Places
	TrustedProxies []*net.IPNet
	LogLevel       string
}

type HTTP struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
	MaxHeaderBytes    int
}

type Database struct {
	URL                string
	MaxConns           int32
	MinConns           int32
	MaxConnLifetime    time.Duration
	MaxConnIdleTime    time.Duration
	HealthCheckPeriod  time.Duration
	QueryTimeout       time.Duration
	TransactionTimeout time.Duration
}

type Redis struct {
	URL          string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type JWT struct {
	Secret   string
	Issuer   string
	Audience string
}

type Places struct {
	PublicCacheTTL    time.Duration
	PublicRateLimit   int64
	AdminRateLimit    int64
	RateLimitWindow   time.Duration
	PlaceFeedHardCap  int
	CollectionHardCap int
}

func Load() (Config, error) {
	cfg := Config{
		Environment: env("APP_ENV", "development"),
		HTTP:        HTTP{Address: env("HTTP_ADDR", ":8080")},
		Database:    Database{URL: os.Getenv("DATABASE_URL")},
		Redis:       Redis{URL: os.Getenv("REDIS_URL")},
		JWT:         JWT{Secret: os.Getenv("JWT_ACCESS_SECRET"), Issuer: env("JWT_ISSUER", "bazgain"), Audience: env("JWT_AUDIENCE", "bazgain-mobile")},
		LogLevel:    env("LOG_LEVEL", "info"),
	}

	var err error
	if cfg.HTTP.ReadHeaderTimeout, err = duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadTimeout, err = duration("HTTP_READ_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.WriteTimeout, err = duration("HTTP_WRITE_TIMEOUT", 30*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.IdleTimeout, err = duration("HTTP_IDLE_TIMEOUT", 60*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ShutdownTimeout, err = duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.MaxBodyBytes, err = int64Value("REQUEST_BODY_MAX_BYTES", 1<<20); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.MaxHeaderBytes, err = intValue("HTTP_MAX_HEADER_BYTES", 32<<10); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxConns, err = int32Value("DB_MAX_CONNS", 10); err != nil {
		return Config{}, err
	}
	if cfg.Database.MinConns, err = int32Value("DB_MIN_CONNS", 1); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxConnLifetime, err = duration("DB_MAX_CONN_LIFETIME", 30*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxConnIdleTime, err = duration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.Database.HealthCheckPeriod, err = duration("DB_HEALTH_CHECK_PERIOD", 30*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.Database.QueryTimeout, err = duration("DB_QUERY_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.Database.TransactionTimeout, err = duration("DB_TX_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.Redis.DialTimeout, err = duration("REDIS_DIAL_TIMEOUT", 2*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.Redis.ReadTimeout, err = duration("REDIS_READ_TIMEOUT", time.Second); err != nil {
		return Config{}, err
	}
	if cfg.Redis.WriteTimeout, err = duration("REDIS_WRITE_TIMEOUT", time.Second); err != nil {
		return Config{}, err
	}

	cacheSeconds, err := int64Value("PUBLIC_CACHE_TTL_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}
	windowSeconds, err := int64Value("RATE_LIMIT_WINDOW_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}
	publicLimit, err := int64Value("PUBLIC_RATE_LIMIT", 60)
	if err != nil {
		return Config{}, err
	}
	adminLimit, err := int64Value("ADMIN_MUTATION_RATE_LIMIT", 30)
	if err != nil {
		return Config{}, err
	}
	placeCap, err := intValue("PLACE_PUBLIC_FEED_HARD_CAP", 1000)
	if err != nil {
		return Config{}, err
	}
	collectionCap, err := intValue("COLLECTIONS_PUBLIC_FEED_HARD_CAP", 100)
	if err != nil {
		return Config{}, err
	}
	cfg.Places = Places{PublicCacheTTL: time.Duration(cacheSeconds) * time.Second, PublicRateLimit: publicLimit, AdminRateLimit: adminLimit, RateLimitWindow: time.Duration(windowSeconds) * time.Second, PlaceFeedHardCap: placeCap, CollectionHardCap: collectionCap}

	if cfg.TrustedProxies, err = parseCIDRs(os.Getenv("TRUSTED_PROXY_CIDRS")); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if c.Database.URL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if _, err := url.ParseRequestURI(c.Database.URL); err != nil {
		return fmt.Errorf("DATABASE_URL: %w", err)
	}
	if c.Redis.URL == "" {
		return errors.New("REDIS_URL is required")
	}
	if _, err := url.ParseRequestURI(c.Redis.URL); err != nil {
		return fmt.Errorf("REDIS_URL: %w", err)
	}
	if len(c.JWT.Secret) < 32 {
		return errors.New("JWT_ACCESS_SECRET must be at least 32 characters")
	}
	if c.JWT.Issuer == "" || c.JWT.Audience == "" {
		return errors.New("JWT issuer and audience are required")
	}
	if c.HTTP.MaxBodyBytes <= 0 || c.HTTP.MaxHeaderBytes < 16<<10 || c.HTTP.MaxHeaderBytes > 32<<10 || c.Database.MaxConns <= 0 || c.Database.MinConns < 0 || c.Database.MinConns > c.Database.MaxConns {
		return errors.New("invalid HTTP or database limits")
	}
	if c.HTTP.ReadHeaderTimeout <= 0 || c.HTTP.ReadTimeout <= 0 || c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 || c.HTTP.ShutdownTimeout <= 0 {
		return errors.New("HTTP timeouts must be positive")
	}
	if c.Database.MaxConnLifetime <= 0 || c.Database.MaxConnIdleTime <= 0 || c.Database.HealthCheckPeriod <= 0 || c.Database.QueryTimeout <= 0 || c.Database.TransactionTimeout <= 0 {
		return errors.New("database timeouts must be positive")
	}
	if c.Redis.DialTimeout <= 0 || c.Redis.ReadTimeout <= 0 || c.Redis.WriteTimeout <= 0 {
		return errors.New("Redis timeouts must be positive")
	}
	if c.Places.PublicRateLimit <= 0 || c.Places.AdminRateLimit <= 0 || c.Places.RateLimitWindow < time.Second || c.Places.PublicCacheTTL <= 0 {
		return errors.New("invalid cache or rate limit settings")
	}
	if c.Places.PlaceFeedHardCap <= 0 || c.Places.CollectionHardCap <= 0 {
		return errors.New("public feed caps must be positive")
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := env(key, fallback.String())
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
func int64Value(key string, fallback int64) (int64, error) {
	value := env(key, strconv.FormatInt(fallback, 10))
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
func int32Value(key string, fallback int32) (int32, error) {
	parsed, err := int64Value(key, int64(fallback))
	if err != nil {
		return 0, err
	}
	if parsed < 0 || parsed > 1<<31-1 {
		return 0, fmt.Errorf("%s is out of range", key)
	}
	return int32(parsed), nil
}
func intValue(key string, fallback int) (int, error) {
	parsed, err := int64Value(key, int64(fallback))
	if err != nil {
		return 0, err
	}
	return int(parsed), nil
}
func parseCIDRs(raw string) ([]*net.IPNet, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	result := make([]*net.IPNet, 0, len(parts))
	for _, part := range parts {
		_, network, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS: %w", err)
		}
		result = append(result, network)
	}
	return result, nil
}
