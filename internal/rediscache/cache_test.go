package rediscache

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/wire"
	"github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1, DialTimeout: 50 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestBuildETagUsesUnixMilliseconds(t *testing.T) {
	cache := NewPlacesCache(nil, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if got := cache.BuildETag(nil); got != `"0"` {
		t.Fatalf("nil ETag = %q", got)
	}
	version := time.UnixMilli(1729000000123)
	if got := cache.BuildETag(&version); got != `"1729000000123"` {
		t.Fatalf("version ETag = %q", got)
	}
}

func TestCollectionsCacheRoundTripAndTTL(t *testing.T) {
	server, client := testRedis(t)
	cache := NewPlacesCache(client, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	version := time.UnixMilli(1729000000123)
	payload := wire.PlaceCollectionsResponse{Collections: []wire.PlaceCollectionSummary{{ID: "c1", Slug: "nature", Title: "Nature", Filters: []wire.PlaceCollectionFilterSummary{}}}}
	if err := cache.SetCollections(context.Background(), payload, &version); err != nil {
		t.Fatalf("SetCollections() error = %v", err)
	}
	got, err := cache.GetCollections(context.Background(), &version)
	if err != nil || got == nil || got.ETag != `"1729000000123"` || len(got.Payload.Collections) != 1 {
		t.Fatalf("GetCollections() = %#v, %v", got, err)
	}
	if ttl := server.TTL(cache.collectionKey(&version)); ttl != time.Minute {
		t.Fatalf("TTL = %v", ttl)
	}
}

func TestListAndDetailCacheRoundTripPreservesEmptyCollectionsAndNulls(t *testing.T) {
	_, client := testRedis(t)
	cache := NewPlacesCache(client, 30*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	version := time.UnixMilli(1729000000456)
	list := wire.PlaceListResponse{Items: []wire.PlaceSummary{{ID: "p1", Summary: nil, Badges: []wire.Badge{}, Hashtags: []wire.PlaceTag{}, CollectionMemberships: []wire.PlaceCollectionMembership{}}}}
	if err := cache.SetPlaceList(context.Background(), list, &version); err != nil {
		t.Fatal(err)
	}
	gotList, err := cache.GetPlaceList(context.Background(), &version)
	if err != nil || gotList == nil || gotList.ETag != `"1729000000456"` || gotList.Payload.Items[0].Badges == nil || gotList.Payload.Items[0].Summary != nil {
		t.Fatalf("list=%#v err=%v", gotList, err)
	}

	detail := wire.PlaceDetailsResponse{Place: wire.PlaceDetails{ID: "p1", Gallery: []wire.MediaAsset{}, Contacts: []wire.PlaceContact{}, Hashtags: []wire.PlaceTag{}, Badges: []wire.Badge{}, InfoBlocks: []wire.PlaceInfoBlock{}}}
	if err := cache.SetPlaceDetail(context.Background(), "valid-slug", detail, &version); err != nil {
		t.Fatal(err)
	}
	gotDetail, err := cache.GetPlaceDetail(context.Background(), "valid-slug", &version)
	if err != nil || gotDetail == nil || gotDetail.Payload.Place.Gallery == nil || gotDetail.Payload.Place.OpeningHours != nil {
		t.Fatalf("detail=%#v err=%v", gotDetail, err)
	}
}

func TestCacheMissAndCorruptPayload(t *testing.T) {
	server, client := testRedis(t)
	cache := NewPlacesCache(client, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	got, err := cache.GetPlaceList(context.Background(), nil)
	if err != nil || got != nil {
		t.Fatalf("cache miss = %#v, %v", got, err)
	}
	server.Set(cache.listKey(nil), "not-json")
	got, err = cache.GetPlaceList(context.Background(), nil)
	if err == nil || got != nil {
		t.Fatalf("corrupt cache = %#v, %v", got, err)
	}
}

func TestDetailCacheRejectsUnsafeSlug(t *testing.T) {
	_, client := testRedis(t)
	cache := NewPlacesCache(client, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := cache.GetPlaceDetail(context.Background(), "../unsafe", nil); err == nil {
		t.Fatal("GetPlaceDetail() expected invalid slug error")
	}
	if err := cache.SetPlaceDetail(context.Background(), "../unsafe", wire.PlaceDetailsResponse{}, nil); err == nil {
		t.Fatal("SetPlaceDetail() expected invalid slug error")
	}
}

func TestRedisFailuresAreReturnedForFailOpenOrFailClosedCallerPolicy(t *testing.T) {
	server, client := testRedis(t)
	cache := NewPlacesCache(client, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	limiter := NewRateLimiter(client)
	server.Close()

	if _, err := cache.GetPlaceList(context.Background(), nil); err == nil {
		t.Fatal("cache read must expose Redis failure so service can fail open")
	}
	if err := cache.SetPlaceList(context.Background(), wire.PlaceListResponse{}, nil); err == nil {
		t.Fatal("cache write must expose Redis failure so service can continue")
	}
	if allowed, _, err := limiter.Allow(context.Background(), "public", "127.0.0.1", 1, time.Minute); err == nil || allowed {
		t.Fatalf("rate limiter must fail closed: allowed=%v err=%v", allowed, err)
	}
	if revoked, err := limiter.SessionRevoked(context.Background(), "session"); err == nil || revoked {
		t.Fatalf("revocation lookup must fail closed at auth layer: revoked=%v err=%v", revoked, err)
	}
}

func TestRateLimiterBoundaryAndRevocation(t *testing.T) {
	server, client := testRedis(t)
	limiter := NewRateLimiter(client)
	for attempt := 1; attempt <= 3; attempt++ {
		allowed, retry, err := limiter.Allow(context.Background(), "public", "127.0.0.1", 2, time.Minute)
		if err != nil {
			t.Fatalf("Allow(%d) error = %v", attempt, err)
		}
		if allowed != (attempt <= 2) {
			t.Fatalf("Allow(%d) = %v", attempt, allowed)
		}
		if retry <= 0 || retry > time.Minute {
			t.Fatalf("Allow(%d) retry = %v", attempt, retry)
		}
	}

	revoked, err := limiter.SessionRevoked(context.Background(), "session")
	if err != nil || revoked {
		t.Fatalf("initial revocation = %v, %v", revoked, err)
	}
	server.Set("auth:session:revoked:session", "1")
	revoked, err = limiter.SessionRevoked(context.Background(), "session")
	if err != nil || !revoked {
		t.Fatalf("stored revocation = %v, %v", revoked, err)
	}
}

func TestRateLimiterRejectsSubSecondWindow(t *testing.T) {
	_, client := testRedis(t)
	limiter := NewRateLimiter(client)
	if allowed, retry, err := limiter.Allow(context.Background(), "public", "127.0.0.1", 1, 500*time.Millisecond); err == nil || allowed || retry != 0 {
		t.Fatalf("Allow() = allowed=%v retry=%v err=%v", allowed, retry, err)
	}
}

func TestRateLimiterTTLTracksNextFixedWindowBoundary(t *testing.T) {
	_, client := testRedis(t)
	limiter := NewRateLimiter(client)
	const window = 17 * time.Second
	before := time.Now().UTC().Unix()
	allowed, retry, err := limiter.Allow(context.Background(), "public", "198.51.100.1", 1, window)
	after := time.Now().UTC().Unix()
	if err != nil || !allowed {
		t.Fatalf("Allow() = allowed=%v retry=%v err=%v", allowed, retry, err)
	}
	valid := false
	for second := before; second <= after; second++ {
		want := time.Duration(17-second%17) * time.Second
		if retry == want {
			valid = true
			break
		}
	}
	if !valid {
		t.Fatalf("retry=%v is not a next-boundary TTL for unix seconds %d..%d", retry, before, after)
	}
}
