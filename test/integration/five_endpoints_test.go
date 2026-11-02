package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/tx-mi/bazgain-places-go/internal/auth"
	"github.com/tx-mi/bazgain-places-go/internal/health"
	"github.com/tx-mi/bazgain-places-go/internal/places"
	postgresstore "github.com/tx-mi/bazgain-places-go/internal/postgres"
	"github.com/tx-mi/bazgain-places-go/internal/rediscache"
	"github.com/tx-mi/bazgain-places-go/internal/transport"
)

const (
	adminID       = "019a0000-0000-7000-8000-000000000001"
	sessionID     = "019a0000-0000-7000-8000-000000000901"
	publishedID   = "019a0000-0000-7000-8000-000000000100"
	publishableID = "019a0000-0000-7000-8000-000000000101"
	noGalleryID   = "019a0000-0000-7000-8000-000000000102"
	noNavID       = "019a0000-0000-7000-8000-000000000103"
	testSecret    = "01234567890123456789012345678901"
)

type environment struct {
	server *httptest.Server
	pool   *pgxpool.Pool
	redis  *redis.Client
}

func startContainer(t *testing.T, request testcontainers.ContainerRequest, port nat.Port) (testcontainers.Container, string) {
	t.Helper()
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: request, Started: true})
	if err != nil {
		t.Fatalf("start %s: %v", request.Image, err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("terminate %s: %v", request.Image, err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	mapped, err := container.MappedPort(ctx, string(port))
	if err != nil {
		t.Fatalf("mapped port %s: %v", port, err)
	}
	return container, host + ":" + mapped.Port()
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func applyLocalSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, name := range []string{"000_postgis.sql", "001_auth.sql", "002_places.sql", "003_seed.sql"} {
		contents, err := os.ReadFile(filepath.Join(repoRoot(t), "db", "local", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(context.Background(), string(contents)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO sessions(id,user_id,created_at,last_seen_at) VALUES($1::uuid,$2::uuid,statement_timestamp(),statement_timestamp()) ON CONFLICT (id) DO NOTHING`, sessionID, adminID); err != nil {
		t.Fatalf("seed admin session: %v", err)
	}
}

func newEnvironment(t *testing.T) *environment {
	t.Helper()
	if os.Getenv("REQUIRE_INTEGRATION") != "1" {
		testcontainers.SkipIfProviderIsNotHealthy(t)
	}
	_, postgresAddress := startContainer(t, testcontainers.ContainerRequest{
		Image:        "postgis/postgis:16-3.4",
		Env:          map[string]string{"POSTGRES_DB": "bazgain_places", "POSTGRES_USER": "bazgain", "POSTGRES_PASSWORD": "bazgain", "TZ": "UTC"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
	}, nat.Port("5432/tcp"))
	_, redisAddress := startContainer(t, testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(time.Minute),
	}, nat.Port("6379/tcp"))

	poolConfig, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://bazgain:bazgain@%s/bazgain_places?sslmode=disable", postgresAddress))
	if err != nil {
		t.Fatalf("parse PostgreSQL URL: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	readyContext, cancelReady := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelReady()
	for {
		if err = pool.Ping(readyContext); err == nil {
			break
		}
		select {
		case <-readyContext.Done():
			t.Fatalf("PostgreSQL did not become ready: %v", err)
		case <-time.After(250 * time.Millisecond):
		}
	}
	applyLocalSchema(t, pool)

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddress})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("Redis did not become ready: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := postgresstore.New(pool, 5*time.Second, 10*time.Second)
	cache := rediscache.NewPlacesCache(redisClient, time.Minute, logger)
	service := places.NewService(repository, cache, logger, 1000, 100)
	limiter := rediscache.NewRateLimiter(redisClient)
	authenticator := auth.New(testSecret, "bazgain", "bazgain-mobile", repository, limiter)
	checker := health.New(pool, redisClient)
	handler := transport.NewServer(service, authenticator, limiter, checker, logger, transport.Options{
		PublicLimit: 1000, AdminLimit: 1000, RateWindow: time.Minute, CacheTTL: time.Minute, MaxBodyBytes: 1 << 20,
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &environment{server: server, pool: pool, redis: redisClient}
}

// NestJS currently allows publishing a gallery that contains only videos. The
// Go migration candidate intentionally preserves that runtime behavior even
// though the public list contract requires a non-empty image URI for the cover.
// This test makes the known parity limitation explicit until the source product
// contract and both implementations can be changed together.
func TestVideoOnlyGalleryPreservesCurrentNestJSPublishParity(t *testing.T) {
	env := newEnvironment(t)
	if _, err := env.pool.Exec(context.Background(), `UPDATE place_media SET kind='video'::place_media_kind WHERE place_id=$1::uuid`, publishableID); err != nil {
		t.Fatalf("make gallery video-only: %v", err)
	}
	response := request(t, env, http.MethodPost, "/v1/admin/places/"+publishableID+"/publish", nil, adminToken(t))
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("publish video-only gallery status=%d body=%s", response.StatusCode, body)
	}
	response = request(t, env, http.MethodGet, "/v1/places", nil, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", response.StatusCode)
	}
	items, ok := decodeJSON(t, response)["items"].([]any)
	if !ok {
		t.Fatalf("items=%#v", items)
	}
	var found map[string]any
	for _, raw := range items {
		item, itemOK := raw.(map[string]any)
		if itemOK && item["id"] == publishableID {
			found = item
			break
		}
	}
	if found == nil || found["cover_image_url"] != "" {
		t.Fatalf("expected documented empty-cover parity behavior, item=%#v", found)
	}
}

func adminToken(t *testing.T) string {
	t.Helper()
	version := 0
	claims := auth.Claims{
		SessionID: sessionID, TokenVersion: &version, Role: "admin", Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: adminID, Issuer: "bazgain", Audience: jwt.ClaimStrings{"bazgain-mobile"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return signed
}

func request(t *testing.T, env *environment, method, path string, body io.Reader, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, env.server.URL+path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func decodeJSON(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatalf("decode JSON status=%d: %v", response.StatusCode, err)
	}
	return value
}

func TestFiveEndpointsAgainstPostGISAndRedis(t *testing.T) {
	env := newEnvironment(t)

	t.Run("collections", func(t *testing.T) {
		response := request(t, env, http.MethodGet, "/v1/place-collections", nil, "")
		if response.StatusCode != http.StatusOK || response.Header.Get("ETag") == "" {
			t.Fatalf("status=%d ETag=%q", response.StatusCode, response.Header.Get("ETag"))
		}
		payload := decodeJSON(t, response)
		collections, ok := payload["collections"].([]any)
		if !ok || len(collections) != 1 {
			t.Fatalf("collections=%#v", payload["collections"])
		}
	})

	t.Run("places list and conditional request", func(t *testing.T) {
		response := request(t, env, http.MethodGet, "/v1/places", nil, "")
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", response.StatusCode)
		}
		etag := response.Header.Get("ETag")
		payload := decodeJSON(t, response)
		items, ok := payload["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("items=%#v", payload["items"])
		}
		req, err := http.NewRequest(http.MethodGet, env.server.URL+"/v1/places", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("If-None-Match", etag)
		conditional, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer conditional.Body.Close()
		if conditional.StatusCode != http.StatusNotModified {
			t.Fatalf("conditional status=%d", conditional.StatusCode)
		}
	})

	t.Run("place detail with PostGIS", func(t *testing.T) {
		response := request(t, env, http.MethodGet, "/v1/places/sulakskiy-kanon", nil, "")
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status=%d", response.StatusCode)
		}
		place, ok := decodeJSON(t, response)["place"].(map[string]any)
		if !ok || place["slug"] != "sulakskiy-kanon" {
			t.Fatalf("place=%#v", place)
		}
		navigation, ok := place["navigation_point"].(map[string]any)
		if !ok || navigation["type"] != "Point" {
			t.Fatalf("navigation_point=%#v", place["navigation_point"])
		}
	})

	token := adminToken(t)
	t.Run("publish", func(t *testing.T) {
		response := request(t, env, http.MethodPost, "/v1/admin/places/"+publishableID+"/publish", bytes.NewBufferString(`{}`), token)
		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("status=%d body=%s", response.StatusCode, body)
		}
		if payload := decodeJSON(t, response); payload["status"] != "published" {
			t.Fatalf("payload=%#v", payload)
		}
	})

	t.Run("archive", func(t *testing.T) {
		response := request(t, env, http.MethodPost, "/v1/admin/places/"+publishedID+"/archive", nil, token)
		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("status=%d body=%s", response.StatusCode, body)
		}
		if payload := decodeJSON(t, response); payload["status"] != "archived" {
			t.Fatalf("payload=%#v", payload)
		}
	})

	t.Run("publish guards", func(t *testing.T) {
		for _, test := range []struct {
			id, code string
		}{{noGalleryID, "cannot_publish_place_without_gallery"}, {noNavID, "cannot_publish_place_without_navigation_point"}} {
			response := request(t, env, http.MethodPost, "/v1/admin/places/"+test.id+"/publish", nil, token)
			if response.StatusCode != http.StatusConflict {
				t.Fatalf("%s status=%d", test.id, response.StatusCode)
			}
			errorBody, ok := decodeJSON(t, response)["error"].(map[string]any)
			if !ok || errorBody["code"] != test.code {
				t.Fatalf("%s error=%#v", test.id, errorBody)
			}
		}
	})

	var auditCount int
	if err := env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM audit_events`).Scan(&auditCount); err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("audit count=%d, want 2", auditCount)
	}
	var nonV7AuditCount int
	if err := env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM audit_events WHERE substring(id::text, 15, 1) <> '7'`).Scan(&nonV7AuditCount); err != nil {
		t.Fatalf("inspect audit UUID versions: %v", err)
	}
	if nonV7AuditCount != 0 {
		t.Fatalf("non-v7 audit IDs=%d", nonV7AuditCount)
	}
}

func TestLifecycleRollbackAndConcurrentCommands(t *testing.T) {
	env := newEnvironment(t)
	token := adminToken(t)

	if _, err := env.pool.Exec(context.Background(), `ALTER TABLE audit_events ADD CONSTRAINT reject_test_audit CHECK (event_type <> 'place_published')`); err != nil {
		t.Fatalf("add audit failure constraint: %v", err)
	}
	response := request(t, env, http.MethodPost, "/v1/admin/places/"+publishableID+"/publish", nil, token)
	if response.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("rollback status=%d body=%s", response.StatusCode, body)
	}
	var status string
	if err := env.pool.QueryRow(context.Background(), `SELECT status::text FROM places WHERE id=$1`, publishableID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "draft" {
		t.Fatalf("failed audit did not roll back status: %q", status)
	}
	if _, err := env.pool.Exec(context.Background(), `ALTER TABLE audit_events DROP CONSTRAINT reject_test_audit`); err != nil {
		t.Fatal(err)
	}

	responses := make(chan int, 2)
	for _, suffix := range []string{"publish", "archive"} {
		go func(suffix string) {
			req, _ := http.NewRequest(http.MethodPost, env.server.URL+"/v1/admin/places/"+publishableID+"/"+suffix, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				responses <- 0
				return
			}
			defer resp.Body.Close()
			responses <- resp.StatusCode
		}(suffix)
	}
	for range 2 {
		if code := <-responses; code != http.StatusOK {
			t.Fatalf("concurrent lifecycle status=%d", code)
		}
	}
	if err := env.pool.QueryRow(context.Background(), `SELECT status::text FROM places WHERE id=$1`, publishableID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "published" && status != "archived" {
		t.Fatalf("final lifecycle status=%q", status)
	}
	var successAudits int
	if err := env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM audit_events WHERE meta->>'place_id'=$1`, publishableID).Scan(&successAudits); err != nil {
		t.Fatal(err)
	}
	if successAudits != 2 {
		t.Fatalf("lifecycle audit count=%d, want 2", successAudits)
	}
}

func TestRedisFailurePolicies(t *testing.T) {
	env := newEnvironment(t)
	if err := env.redis.Close(); err != nil {
		t.Fatal(err)
	}

	// The distributed limiter is fail-closed before the public cache is reached.
	response, err := http.Get(env.server.URL + "/v1/places")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("rate limiter failure status=%d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}
}
