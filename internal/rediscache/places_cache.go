package rediscache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/mgmd-abdulaev/bazgain-places-go/internal/places"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/wire"
	"github.com/redis/go-redis/v9"
)

var safeSlug = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,118}[a-z0-9])?$`)

type PlacesCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *slog.Logger
}

func NewPlacesCache(client *redis.Client, ttl time.Duration, logger *slog.Logger) *PlacesCache {
	return &PlacesCache{client: client, ttl: ttl, logger: logger}
}

func (c *PlacesCache) BuildETag(version *time.Time) string {
	var ms int64
	if version != nil {
		ms = version.UnixMilli()
	}
	return fmt.Sprintf("\"%d\"", ms)
}
func (c *PlacesCache) version(version *time.Time) int64 {
	if version == nil {
		return 0
	}
	return version.UnixMilli()
}
func (c *PlacesCache) collectionKey(v *time.Time) string {
	return fmt.Sprintf("places:public:collections:v1:%d", c.version(v))
}
func (c *PlacesCache) listKey(v *time.Time) string {
	return fmt.Sprintf("places:public:list:v1:%d", c.version(v))
}
func (c *PlacesCache) detailKey(slug string, v *time.Time) (string, error) {
	if !safeSlug.MatchString(slug) {
		return "", fmt.Errorf("invalid slug")
	}
	return fmt.Sprintf("places:public:detail:v1:%s:%d", slug, c.version(v)), nil
}

func read[T any](ctx context.Context, c *PlacesCache, key, etag string) (*places.Cached[T], error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		c.logger.WarnContext(ctx, "redis GET failed; treating as cache miss", "key", key, "error", err)
		return nil, err
	}
	var payload T
	if err := json.Unmarshal(raw, &payload); err != nil {
		c.logger.WarnContext(ctx, "cached payload invalid; treating as cache miss", "key", key, "error", err)
		return nil, err
	}
	return &places.Cached[T]{ETag: etag, Payload: payload}, nil
}
func write[T any](ctx context.Context, c *PlacesCache, key string, payload T) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := c.client.Set(ctx, key, raw, c.ttl).Err(); err != nil {
		c.logger.WarnContext(ctx, "redis SET failed; continuing without cache", "key", key, "error", err)
		return err
	}
	return nil
}

func (c *PlacesCache) GetCollections(ctx context.Context, v *time.Time) (*places.Cached[wire.PlaceCollectionsResponse], error) {
	return read[wire.PlaceCollectionsResponse](ctx, c, c.collectionKey(v), c.BuildETag(v))
}
func (c *PlacesCache) SetCollections(ctx context.Context, p wire.PlaceCollectionsResponse, v *time.Time) error {
	return write(ctx, c, c.collectionKey(v), p)
}
func (c *PlacesCache) GetPlaceList(ctx context.Context, v *time.Time) (*places.Cached[wire.PlaceListResponse], error) {
	return read[wire.PlaceListResponse](ctx, c, c.listKey(v), c.BuildETag(v))
}
func (c *PlacesCache) SetPlaceList(ctx context.Context, p wire.PlaceListResponse, v *time.Time) error {
	return write(ctx, c, c.listKey(v), p)
}
func (c *PlacesCache) GetPlaceDetail(ctx context.Context, slug string, v *time.Time) (*places.Cached[wire.PlaceDetailsResponse], error) {
	key, err := c.detailKey(slug, v)
	if err != nil {
		return nil, err
	}
	return read[wire.PlaceDetailsResponse](ctx, c, key, c.BuildETag(v))
}
func (c *PlacesCache) SetPlaceDetail(ctx context.Context, slug string, p wire.PlaceDetailsResponse, v *time.Time) error {
	key, err := c.detailKey(slug, v)
	if err != nil {
		return err
	}
	return write(ctx, c, key, p)
}
