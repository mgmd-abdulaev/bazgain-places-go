package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/tx-mi/bazgain-places-go/internal/places"
	"github.com/tx-mi/bazgain-places-go/internal/wire"
)

type transportRepo struct {
	version         *time.Time
	collections     []places.CollectionRecord
	places          []places.PlaceRecord
	detail          *places.PlaceRecord
	admin           *places.PlaceRecord
	err             error
	publishID       string
	archiveID       string
	publishActor    places.Actor
	archiveActor    places.Actor
	publishEnvelope places.AuditEnvelope
	archiveEnvelope places.AuditEnvelope
}

func (r *transportRepo) CollectionsVersion(context.Context) (*time.Time, error) {
	return r.version, r.err
}
func (r *transportRepo) ListPublishedCollections(context.Context, int) ([]places.CollectionRecord, error) {
	return r.collections, r.err
}
func (r *transportRepo) PlacesVersion(context.Context) (*time.Time, error) { return r.version, r.err }
func (r *transportRepo) ListPublishedPlaces(context.Context, int) ([]places.PlaceRecord, error) {
	return r.places, r.err
}
func (r *transportRepo) DetailVersion(context.Context, string) (*time.Time, error) {
	return r.version, r.err
}
func (r *transportRepo) FindPublishedPlaceBySlug(context.Context, string) (*places.PlaceRecord, error) {
	return r.detail, r.err
}
func (r *transportRepo) Publish(_ context.Context, id string, actor places.Actor, envelope places.AuditEnvelope) (*places.PlaceRecord, error) {
	r.publishID, r.publishActor, r.publishEnvelope = id, actor, envelope
	return r.admin, r.err
}
func (r *transportRepo) Archive(_ context.Context, id string, actor places.Actor, envelope places.AuditEnvelope) (*places.PlaceRecord, error) {
	r.archiveID, r.archiveActor, r.archiveEnvelope = id, actor, envelope
	return r.admin, r.err
}

type transportCache struct{}

func (transportCache) BuildETag(*time.Time) string { return `"0"` }
func (transportCache) GetCollections(context.Context, *time.Time) (*places.Cached[wire.PlaceCollectionsResponse], error) {
	return nil, nil
}
func (transportCache) SetCollections(context.Context, wire.PlaceCollectionsResponse, *time.Time) error {
	return nil
}
func (transportCache) GetPlaceList(context.Context, *time.Time) (*places.Cached[wire.PlaceListResponse], error) {
	return nil, nil
}
func (transportCache) SetPlaceList(context.Context, wire.PlaceListResponse, *time.Time) error {
	return nil
}
func (transportCache) GetPlaceDetail(context.Context, string, *time.Time) (*places.Cached[wire.PlaceDetailsResponse], error) {
	return nil, nil
}
func (transportCache) SetPlaceDetail(context.Context, string, wire.PlaceDetailsResponse, *time.Time) error {
	return nil
}

type authStub struct {
	actor places.Actor
	err   error
}

func (a authStub) Authenticate(context.Context, string) (places.Actor, error) { return a.actor, a.err }

type limiterStub struct {
	allowed bool
	retry   time.Duration
	err     error
}

func (l limiterStub) Allow(context.Context, string, string, int64, time.Duration) (bool, time.Duration, error) {
	return l.allowed, l.retry, l.err
}

type healthStub struct{ postgresErr, redisErr error }

func (h healthStub) Postgres(context.Context) error { return h.postgresErr }
func (h healthStub) Redis(context.Context) error    { return h.redisErr }

func newTestServer(repo *transportRepo, auth Authenticator, limiter Limiter, health Health, maxBody int64) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := places.NewService(repo, transportCache{}, logger, 1000, 100)
	return NewServer(service, auth, limiter, health, logger, Options{PublicLimit: 60, AdminLimit: 30, RateWindow: time.Minute, CacheTTL: time.Minute, MaxBodyBytes: maxBody})
}

func transportPlace(status string) places.PlaceRecord {
	now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	return places.PlaceRecord{
		ID: "abcdefab-cdef-4abc-8abc-abcdefabcdef", Slug: "empty-place", Title: "Empty Place", Status: status,
		Kind:            places.KindRecord{ID: "kind-id", Key: "nature", Label: "Nature"},
		NavigationPoint: []byte(`{"type":"Point","coordinates":[46.8,43]}`),
		CreatedAt:       now, UpdatedAt: now,
	}
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) wire.ErrorEnvelope {
	t.Helper()
	var envelope wire.ErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error response %q: %v", recorder.Body.String(), err)
	}
	return envelope
}

func TestCollectionsEndpointReturnsContractShapeAndCacheHeaders(t *testing.T) {
	handler := newTestServer(&transportRepo{}, authStub{}, limiterStub{allowed: true}, healthStub{}, 1024)
	req := httptest.NewRequest(http.MethodGet, "/v1/place-collections", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("ETag") != `"0"` || rec.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("headers=%v", rec.Header())
	}
	if strings.TrimSpace(rec.Body.String()) != `{"collections":[]}` {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestFiveEndpointWireShapesPreserveEmptyArraysNullsAndSnakeCase(t *testing.T) {
	now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	place := transportPlace("published")
	repo := &transportRepo{
		version: &now,
		collections: []places.CollectionRecord{{
			ID: "collection-id", Slug: "empty", Title: "Empty", Filters: []places.CollectionFilterRecord{},
		}},
		places: []places.PlaceRecord{place}, detail: &place,
	}
	handler := newTestServer(repo, authStub{actor: places.Actor{UserID: "admin", SessionID: "session", Role: "admin"}}, limiterStub{allowed: true}, healthStub{}, 1024)

	tests := []struct {
		name, method, path, want string
	}{
		{
			name: "collections", method: http.MethodGet, path: "/v1/place-collections",
			want: `{"collections":[{"id":"collection-id","slug":"empty","title":"Empty","subtitle":null,"description":null,"preview_image_url":null,"icon_image_url":null,"background_color":null,"sort_order":0,"places_count":0,"filters":[]}]}`,
		},
		{
			name: "list", method: http.MethodGet, path: "/v1/places",
			want: `{"items":[{"id":"abcdefab-cdef-4abc-8abc-abcdefabcdef","slug":"empty-place","title":"Empty Place","kind":{"key":"nature","label":"Nature"},"summary":null,"sort_order":0,"cover_image_url":"","location_label":null,"navigation_point":{"type":"Point","coordinates":[46.8,43]},"badges":[],"hashtags":[],"collection_memberships":[],"opening_hours":null}]}`,
		},
		{
			name: "detail", method: http.MethodGet, path: "/v1/places/empty-place",
			want: `{"place":{"id":"abcdefab-cdef-4abc-8abc-abcdefabcdef","slug":"empty-place","title":"Empty Place","kind":{"key":"nature","label":"Nature"},"summary":null,"description_markdown":null,"gallery":[],"geometry":{"type":"Point","coordinates":[46.8,43]},"navigation_point":{"type":"Point","coordinates":[46.8,43]},"address":null,"contacts":[],"hashtags":[],"badges":[],"info_blocks":[],"opening_hours":null}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != test.want {
				t.Fatalf("status=%d\n got=%s\nwant=%s", recorder.Code, strings.TrimSpace(recorder.Body.String()), test.want)
			}
		})
	}

	for _, lifecycle := range []struct {
		name, suffix, status string
	}{{"publish", "publish", "published"}, {"archive", "archive", "archived"}} {
		t.Run(lifecycle.name, func(t *testing.T) {
			admin := transportPlace(lifecycle.status)
			repo.admin = &admin
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/v1/admin/places/ABCDEFAB-CDEF-4ABC-8ABC-ABCDEFABCDEF/"+lifecycle.suffix, nil)
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			body := recorder.Body.String()
			for _, fragment := range []string{`"status":"` + lifecycle.status + `"`, `"summary":null`, `"gallery":[]`, `"contacts":[]`, `"collection_memberships":[]`, `"cover_media_id":null`, `"published_at":null`} {
				if !strings.Contains(body, fragment) {
					t.Fatalf("response missing %s: %s", fragment, body)
				}
			}
			if strings.Contains(body, "collectionMemberships") || strings.Contains(body, "coverMediaId") {
				t.Fatalf("camelCase leaked onto wire: %s", body)
			}
		})
	}
}

func TestAdminEndpointsCanonicalizeIDAndCaptureAuditContext(t *testing.T) {
	record := transportPlace("published")
	repo := &transportRepo{admin: &record}
	actor := places.Actor{UserID: "admin", SessionID: "session", Role: "admin", TokenVersion: 3}
	handler := newTestServer(repo, authStub{actor: actor}, limiterStub{allowed: true}, healthStub{}, 1024)
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/places/ABCDEFAB-CDEF-4ABC-8ABC-ABCDEFABCDEF/publish", nil)
	request.RemoteAddr = "203.0.113.8:4321"
	request.Header.Set("User-Agent", strings.Repeat("x", 600))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.publishID != "abcdefab-cdef-4abc-8abc-abcdefabcdef" || repo.publishActor != actor {
		t.Fatalf("id=%q actor=%#v", repo.publishID, repo.publishActor)
	}
	if repo.publishEnvelope.IP == nil || *repo.publishEnvelope.IP != "203.0.113.8" || repo.publishEnvelope.UserAgent == nil || len(*repo.publishEnvelope.UserAgent) != 512 {
		t.Fatalf("audit envelope=%#v", repo.publishEnvelope)
	}
}

func TestAuditUserAgentIsSanitizedAndTruncatedAtUTF8Boundary(t *testing.T) {
	record := transportPlace("published")
	repo := &transportRepo{admin: &record}
	handler := newTestServer(repo, authStub{actor: places.Actor{Role: "admin"}}, limiterStub{allowed: true}, healthStub{}, 1024)
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/places/abcdefab-cdef-4abc-8abc-abcdefabcdef/publish", nil)
	request.Header["User-Agent"] = []string{strings.Repeat("x", 510) + "\x00\n" + "я" + strings.Repeat("z", 20)}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.publishEnvelope.UserAgent == nil {
		t.Fatal("missing user agent")
	}
	got := *repo.publishEnvelope.UserAgent
	if !utf8.ValidString(got) || len(got) > 512 || strings.ContainsAny(got, "\x00\n\r") {
		t.Fatalf("unsafe stored user agent %q (bytes=%d)", got, len(got))
	}
}

func TestUnknownRouteAndWrongMethodUseJSONErrorEnvelope(t *testing.T) {
	handler := newTestServer(&transportRepo{}, authStub{}, limiterStub{allowed: true}, healthStub{}, 1024)
	tests := []struct {
		name, method, path, code, allow string
		status                          int
	}{
		{name: "unknown route", method: http.MethodGet, path: "/v1/unknown", status: http.StatusNotFound, code: "not_found"},
		{name: "wrong method", method: http.MethodDelete, path: "/v1/places", status: http.StatusMethodNotAllowed, code: "method_not_allowed", allow: http.MethodGet},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			envelope := decodeError(t, recorder)
			if recorder.Code != test.status || envelope.Error.Code != test.code || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") || recorder.Header().Get("Allow") != test.allow {
				t.Fatalf("status=%d allow=%q envelope=%#v headers=%v", recorder.Code, recorder.Header().Get("Allow"), envelope, recorder.Header())
			}
		})
	}
}

func TestPublicEndpointErrorsUseStableEnvelope(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name, path string
		repo       *transportRepo
		status     int
		code       string
	}{
		{"detail not found", "/v1/places/missing", &transportRepo{}, 404, "place_not_found"},
		{"invalid detail slug", "/v1/places/Bad%20Slug", &transportRepo{}, 404, "place_not_found"},
		{"repository error", "/v1/places", &transportRepo{version: &now, err: errors.New("database down")}, 500, "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestServer(test.repo, authStub{}, limiterStub{allowed: true}, healthStub{}, 1024)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			envelope := decodeError(t, recorder)
			if recorder.Code != test.status || envelope.Error.Code != test.code || envelope.Error.Message == "" {
				t.Fatalf("status=%d envelope=%#v", recorder.Code, envelope)
			}
		})
	}
}

func TestPublicEndpointReturns304ForWeakOrListedETag(t *testing.T) {
	handler := newTestServer(&transportRepo{}, authStub{}, limiterStub{allowed: true}, healthStub{}, 1024)
	for _, header := range []string{`W/"0"`, `"other", "0"`, `*`} {
		req := httptest.NewRequest(http.MethodGet, "/v1/places", nil)
		req.Header.Set("If-None-Match", header)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
			t.Fatalf("header=%q status=%d body=%q", header, rec.Code, rec.Body.String())
		}
	}
}

func TestLimiterFailureAndLimitUseStableErrors(t *testing.T) {
	for name, limiter := range map[string]limiterStub{
		"dependency failure": {err: errors.New("redis down")},
		"limited":            {allowed: false, retry: 1500 * time.Millisecond},
	} {
		t.Run(name, func(t *testing.T) {
			handler := newTestServer(&transportRepo{}, authStub{}, limiter, healthStub{}, 1024)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/places", nil))
			envelope := decodeError(t, rec)
			if name == "dependency failure" {
				if rec.Code != 500 || envelope.Error.Code != "internal_error" {
					t.Fatalf("status=%d envelope=%#v", rec.Code, envelope)
				}
			} else {
				if rec.Code != 429 || envelope.Error.Code != "rate_limited" || rec.Header().Get("Retry-After") != "2" {
					t.Fatalf("status=%d retry=%q envelope=%#v", rec.Code, rec.Header().Get("Retry-After"), envelope)
				}
			}
		})
	}
}

func TestAdminRouteValidatesIDAfterAuthentication(t *testing.T) {
	handler := newTestServer(&transportRepo{}, authStub{actor: places.Actor{Role: "admin"}}, limiterStub{allowed: true}, healthStub{}, 1024)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/admin/places/not-a-uuid/publish", nil))
	envelope := decodeError(t, rec)
	if rec.Code != 422 || envelope.Error.Code != "validation_error" || len(envelope.Error.Fields) != 1 || envelope.Error.Fields[0].Code != "invalid_uuid" {
		t.Fatalf("status=%d envelope=%#v", rec.Code, envelope)
	}
}

func TestAdminRoutePropagatesAuthenticationError(t *testing.T) {
	handler := newTestServer(&transportRepo{}, authStub{err: places.Unauthorized()}, limiterStub{allowed: true}, healthStub{}, 1024)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/admin/places/abcdefab-cdef-4abc-8abc-abcdefabcdef/archive", nil))
	if rec.Code != 401 || decodeError(t, rec).Error.Code != "access_token_invalid" {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminEndpointErrorsUseStableEnvelope(t *testing.T) {
	id := "abcdefab-cdef-4abc-8abc-abcdefabcdef"
	tests := []struct {
		name, suffix string
		repo         *transportRepo
		status       int
		code         string
	}{
		{"publish guard", "publish", &transportRepo{err: places.CannotPublishWithoutGallery()}, http.StatusConflict, "cannot_publish_place_without_gallery"},
		{"publish missing", "publish", &transportRepo{}, http.StatusNotFound, "place_not_found"},
		{"archive missing", "archive", &transportRepo{}, http.StatusNotFound, "place_not_found"},
		{"archive internal", "archive", &transportRepo{err: errors.New("database down")}, http.StatusInternalServerError, "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestServer(test.repo, authStub{actor: places.Actor{Role: "admin"}}, limiterStub{allowed: true}, healthStub{}, 1024)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/admin/places/"+id+"/"+test.suffix, nil))
			envelope := decodeError(t, recorder)
			if recorder.Code != test.status || envelope.Error.Code != test.code {
				t.Fatalf("status=%d envelope=%#v", recorder.Code, envelope)
			}
		})
	}
}

func TestAdminRouteIgnoresSmallBodyAndRejectsOversizedBody(t *testing.T) {
	id := "abcdefab-cdef-4abc-8abc-abcdefabcdef"
	record := &transportRepo{admin: func() *places.PlaceRecord {
		now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
		return &places.PlaceRecord{ID: id, Slug: "place", Title: "Place", Status: "archived", Kind: places.KindRecord{ID: "kind", Key: "nature", Label: "Nature"}, CreatedAt: now, UpdatedAt: now}
	}()}
	handler := newTestServer(record, authStub{actor: places.Actor{Role: "admin"}}, limiterStub{allowed: true}, healthStub{}, 4)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/admin/places/"+id+"/archive", strings.NewReader("{}")))
	if rec.Code != 200 {
		t.Fatalf("small body status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/admin/places/"+id+"/archive", strings.NewReader("12345")))
	if rec.Code != 413 || decodeError(t, rec).Error.Code != "payload_too_large" {
		t.Fatalf("large body status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHealthEndpoints(t *testing.T) {
	handler := newTestServer(&transportRepo{}, authStub{}, limiterStub{allowed: true}, healthStub{}, 1024)
	for _, path := range []string{"/v1/health/live", "/v1/health/ready"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
			t.Fatalf("%s status=%d body=%q", path, rec.Code, rec.Body.String())
		}
	}

	unhealthy := newTestServer(&transportRepo{}, authStub{}, limiterStub{allowed: true}, healthStub{postgresErr: errors.New("down")}, 1024)
	rec := httptest.NewRecorder()
	unhealthy.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/health/ready", nil))
	if rec.Code != 503 {
		t.Fatalf("unhealthy status=%d", rec.Code)
	}
}

func TestClientIPTrustsForwardingOnlyFromConfiguredProxy(t *testing.T) {
	_, trustedNetwork, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	if got := clientIP(req, []*net.IPNet{trustedNetwork}); got != "203.0.113.10" {
		t.Fatalf("untrusted peer yielded %q", got)
	}

	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.3")
	if got := clientIP(req, []*net.IPNet{trustedNetwork}); got != "198.51.100.7" {
		t.Fatalf("trusted proxy chain yielded %q", got)
	}

	req.Header.Set("X-Forwarded-For", strings.Repeat("1", 4097))
	if got := clientIP(req, []*net.IPNet{trustedNetwork}); got != "10.0.0.2" {
		t.Fatalf("oversized forwarded chain must fall back to peer, got %q", got)
	}
}
