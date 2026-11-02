package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Checker struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func New(db *pgxpool.Pool, redisClient *redis.Client) *Checker {
	return &Checker{db: db, redis: redisClient}
}
func (c *Checker) Postgres(ctx context.Context) error { return c.db.Ping(ctx) }
func (c *Checker) Redis(ctx context.Context) error    { return c.redis.Ping(ctx).Err() }
