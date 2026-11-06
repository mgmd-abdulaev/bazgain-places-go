package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/auth"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/config"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/health"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/places"
	postgresstore "github.com/mgmd-abdulaev/bazgain-places-go/internal/postgres"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/rediscache"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/transport"
	"github.com/redis/go-redis/v9"
)

func Run(ctx context.Context, cfg config.Config) error {
	logger := newLogger(cfg.LogLevel)
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("parse database config: %w", err)
	}
	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Database.HealthCheckPeriod
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	startupDBCtx, cancelDBPing := context.WithTimeout(ctx, cfg.Database.QueryTimeout)
	err = pool.Ping(startupDBCtx)
	cancelDBPing()
	if err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	redisOptions, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("parse redis config: %w", err)
	}
	redisOptions.DialTimeout = cfg.Redis.DialTimeout
	redisOptions.ReadTimeout = cfg.Redis.ReadTimeout
	redisOptions.WriteTimeout = cfg.Redis.WriteTimeout
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()
	startupRedisCtx, cancelRedisPing := context.WithTimeout(ctx, cfg.Redis.DialTimeout+cfg.Redis.ReadTimeout)
	err = redisClient.Ping(startupRedisCtx).Err()
	cancelRedisPing()
	if err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	repo := postgresstore.New(pool, cfg.Database.QueryTimeout, cfg.Database.TransactionTimeout)
	cache := rediscache.NewPlacesCache(redisClient, cfg.Places.PublicCacheTTL, logger)
	service := places.NewService(repo, cache, logger, cfg.Places.PlaceFeedHardCap, cfg.Places.CollectionHardCap)
	limiter := rediscache.NewRateLimiter(redisClient)
	authenticator := auth.New(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.Audience, repo, limiter)
	checker := health.New(pool, redisClient)
	handler := transport.NewServer(service, authenticator, limiter, checker, logger, transport.Options{TrustedProxies: cfg.TrustedProxies, PublicLimit: cfg.Places.PublicRateLimit, AdminLimit: cfg.Places.AdminRateLimit, RateWindow: cfg.Places.RateLimitWindow, CacheTTL: cfg.Places.PublicCacheTTL, MaxBodyBytes: cfg.HTTP.MaxBodyBytes})
	server := &http.Server{Addr: cfg.HTTP.Address, Handler: handler, ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout, ReadTimeout: cfg.HTTP.ReadTimeout, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout, MaxHeaderBytes: cfg.HTTP.MaxHeaderBytes}
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "address", cfg.HTTP.Address)
		serveErr <- server.ListenAndServe()
	}()
	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-shutdownCtx.Done():
		logger.Info("shutdown signal received")
		timeoutCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(timeoutCtx)
	case err := <-serveErr:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	switch level {
	case "debug":
		parsed = slog.LevelDebug
	case "warn":
		parsed = slog.LevelWarn
	case "error":
		parsed = slog.LevelError
	default:
		parsed = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsed}))
}
