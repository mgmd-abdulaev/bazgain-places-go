package places

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/mgmd-abdulaev/bazgain-places-go/internal/wire"
)

type repoStub struct {
	collectionsVersion *time.Time
	collections        []CollectionRecord
	placesVersion      *time.Time
	places             []PlaceRecord
	detailVersion      *time.Time
	detail             *PlaceRecord
	published          *PlaceRecord
	archived           *PlaceRecord
	err                error
	collectionsCalls   int
	listCalls          int
	detailCalls        int
	publishID          string
	archiveID          string
	versionErr         error
	collectionsErr     error
	listErr            error
	detailVersionErr   error
	detailErr          error
	publishErr         error
	archiveErr         error
}

func (r *repoStub) CollectionsVersion(context.Context) (*time.Time, error) {
	if r.versionErr != nil {
		return nil, r.versionErr
	}
	return r.collectionsVersion, r.err
}
func (r *repoStub) ListPublishedCollections(context.Context, int) ([]CollectionRecord, error) {
	r.collectionsCalls++
	if r.collectionsErr != nil {
		return nil, r.collectionsErr
	}
	return r.collections, r.err
}
func (r *repoStub) PlacesVersion(context.Context) (*time.Time, error) {
	if r.versionErr != nil {
		return nil, r.versionErr
	}
	return r.placesVersion, r.err
}
func (r *repoStub) ListPublishedPlaces(context.Context, int) ([]PlaceRecord, error) {
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.places, r.err
}
func (r *repoStub) DetailVersion(context.Context, string) (*time.Time, error) {
	if r.detailVersionErr != nil {
		return nil, r.detailVersionErr
	}
	return r.detailVersion, r.err
}
func (r *repoStub) FindPublishedPlaceBySlug(context.Context, string) (*PlaceRecord, error) {
	r.detailCalls++
	if r.detailErr != nil {
		return nil, r.detailErr
	}
	return r.detail, r.err
}
func (r *repoStub) Publish(_ context.Context, id string, _ Actor, _ AuditEnvelope) (*PlaceRecord, error) {
	r.publishID = id
	if r.publishErr != nil {
		return nil, r.publishErr
	}
	return r.published, r.err
}
func (r *repoStub) Archive(_ context.Context, id string, _ Actor, _ AuditEnvelope) (*PlaceRecord, error) {
	r.archiveID = id
	if r.archiveErr != nil {
		return nil, r.archiveErr
	}
	return r.archived, r.err
}

type cacheStub struct {
	collections *Cached[wire.PlaceCollectionsResponse]
	list        *Cached[wire.PlaceListResponse]
	detail      *Cached[wire.PlaceDetailsResponse]
	getErr      error
	setErr      error
	sets        int
}

func (*cacheStub) BuildETag(v *time.Time) string {
	if v == nil {
		return `"0"`
	}
	return `"version"`
}
func (c *cacheStub) GetCollections(context.Context, *time.Time) (*Cached[wire.PlaceCollectionsResponse], error) {
	return c.collections, c.getErr
}
func (c *cacheStub) SetCollections(context.Context, wire.PlaceCollectionsResponse, *time.Time) error {
	c.sets++
	return c.setErr
}
func (c *cacheStub) GetPlaceList(context.Context, *time.Time) (*Cached[wire.PlaceListResponse], error) {
	return c.list, c.getErr
}
func (c *cacheStub) SetPlaceList(context.Context, wire.PlaceListResponse, *time.Time) error {
	c.sets++
	return c.setErr
}
func (c *cacheStub) GetPlaceDetail(context.Context, string, *time.Time) (*Cached[wire.PlaceDetailsResponse], error) {
	return c.detail, c.getErr
}
func (c *cacheStub) SetPlaceDetail(context.Context, string, wire.PlaceDetailsResponse, *time.Time) error {
	c.sets++
	return c.setErr
}

func testService(repo Repository, cache Cache) *Service {
	return NewService(repo, cache, slog.New(slog.NewTextHandler(io.Discard, nil)), 1000, 100)
}

func validRecord() PlaceRecord {
	now := time.Date(2026, 10, 31, 10, 11, 12, 123000000, time.UTC)
	return PlaceRecord{
		ID: "019a3ee8-f95c-7000-8000-000000000001", Slug: "sulakskiy-kanon", Title: "Сулакский каньон",
		Kind:   KindRecord{ID: "019a3ee8-f95c-7000-8000-000000000002", Key: "nature", Label: "Природа"},
		Status: "published", NavigationPoint: []byte(`{"type":"Point","coordinates":[46.8,43.0]}`),
		CreatedAt: now, UpdatedAt: now,
	}
}

func TestCollectionsEmptyUsesNonNilArrayAndCaches(t *testing.T) {
	repo := &repoStub{}
	cache := &cacheStub{}
	got, err := testService(repo, cache).Collections(context.Background())
	if err != nil {
		t.Fatalf("Collections() error = %v", err)
	}
	if got.ETag != `"0"` || got.Payload.Collections == nil || len(got.Payload.Collections) != 0 {
		t.Fatalf("unexpected empty result: %#v", got)
	}
	if repo.collectionsCalls != 0 || cache.sets != 1 {
		t.Fatalf("repository calls=%d cache sets=%d", repo.collectionsCalls, cache.sets)
	}
}

func TestCollectionsCacheHitSkipsRepository(t *testing.T) {
	now := time.Now().UTC()
	want := wire.PlaceCollectionsResponse{Collections: []wire.PlaceCollectionSummary{{ID: "c1", Slug: "nature", Title: "Природа", Filters: []wire.PlaceCollectionFilterSummary{}}}}
	repo := &repoStub{collectionsVersion: &now}
	cache := &cacheStub{collections: &Cached[wire.PlaceCollectionsResponse]{ETag: `"cached"`, Payload: want}}
	got, err := testService(repo, cache).Collections(context.Background())
	if err != nil {
		t.Fatalf("Collections() error = %v", err)
	}
	if got.ETag != `"cached"` || len(got.Payload.Collections) != 1 || repo.collectionsCalls != 0 {
		t.Fatalf("cache hit not preserved: %#v calls=%d", got, repo.collectionsCalls)
	}
}

func TestCollectionsCacheFailureFallsBackToRepository(t *testing.T) {
	now := time.Now().UTC()
	repo := &repoStub{collectionsVersion: &now, collections: []CollectionRecord{{ID: "c1", Slug: "nature", Title: "Природа", Filters: []CollectionFilterRecord{}}}}
	cache := &cacheStub{getErr: errors.New("redis unavailable"), setErr: errors.New("redis unavailable")}
	got, err := testService(repo, cache).Collections(context.Background())
	if err != nil {
		t.Fatalf("Collections() error = %v", err)
	}
	if len(got.Payload.Collections) != 1 || repo.collectionsCalls != 1 {
		t.Fatalf("database fallback not used: %#v calls=%d", got, repo.collectionsCalls)
	}
}

func TestListEmptyUsesNonNilArray(t *testing.T) {
	got, err := testService(&repoStub{}, &cacheStub{}).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got.Payload.Items == nil || len(got.Payload.Items) != 0 {
		t.Fatalf("items must encode as []: %#v", got.Payload.Items)
	}
}

func TestListCacheHitAndCacheFailureFallback(t *testing.T) {
	now := time.Now().UTC()
	cached := wire.PlaceListResponse{Items: []wire.PlaceSummary{{ID: "cached"}}}
	repo := &repoStub{placesVersion: &now, places: []PlaceRecord{validRecord()}}
	got, err := testService(repo, &cacheStub{list: &Cached[wire.PlaceListResponse]{ETag: `"cached"`, Payload: cached}}).List(context.Background())
	if err != nil || got.ETag != `"cached"` || got.Payload.Items[0].ID != "cached" || repo.listCalls != 0 {
		t.Fatalf("cache result=%#v err=%v calls=%d", got, err, repo.listCalls)
	}

	repo = &repoStub{placesVersion: &now, places: []PlaceRecord{validRecord()}}
	cache := &cacheStub{getErr: errors.New("redis down"), setErr: errors.New("redis down")}
	got, err = testService(repo, cache).List(context.Background())
	if err != nil || len(got.Payload.Items) != 1 || got.Payload.Items[0].Slug != "sulakskiy-kanon" || repo.listCalls != 1 || cache.sets != 1 {
		t.Fatalf("fallback result=%#v err=%v calls=%d sets=%d", got, err, repo.listCalls, cache.sets)
	}
}

func TestListConvertsRepositoryAndMappingFailuresToInternalError(t *testing.T) {
	now := time.Now().UTC()
	bad := validRecord()
	bad.NavigationPoint = []byte(`{"type":"Polygon","coordinates":[]}`)
	for name, repo := range map[string]*repoStub{
		"version": {versionErr: errors.New("version failed")},
		"list":    {placesVersion: &now, listErr: errors.New("list failed")},
		"mapping": {placesVersion: &now, places: []PlaceRecord{bad}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := testService(repo, &cacheStub{}).List(context.Background())
			var appErr *AppError
			if !errors.As(err, &appErr) || appErr.Code != "internal_error" || appErr.Cause == nil {
				t.Fatalf("error=%#v", err)
			}
		})
	}
}

func TestDetailRejectsInvalidAndUUIDSlugsBeforeRepository(t *testing.T) {
	for _, slug := range []string{"Bad Slug", "019a3ee8-f95c-7000-8000-000000000001"} {
		repo := &repoStub{}
		_, err := testService(repo, &cacheStub{}).Detail(context.Background(), slug)
		var appErr *AppError
		if !errors.As(err, &appErr) || appErr.Code != "place_not_found" {
			t.Fatalf("Detail(%q) error = %#v", slug, err)
		}
		if repo.detailCalls != 0 {
			t.Fatalf("Detail(%q) queried repository", slug)
		}
	}
}

func TestDetailReturnsNotFoundWhenRowDisappearsAfterVersionRead(t *testing.T) {
	now := time.Now().UTC()
	repo := &repoStub{detailVersion: &now, detail: nil}
	_, err := testService(repo, &cacheStub{}).Detail(context.Background(), "valid-slug")
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "place_not_found" || repo.detailCalls != 1 {
		t.Fatalf("unexpected race result: err=%#v calls=%d", err, repo.detailCalls)
	}
}

func TestDetailCacheHitAndCacheFailureFallback(t *testing.T) {
	now := time.Now().UTC()
	cached := wire.PlaceDetailsResponse{Place: wire.PlaceDetails{ID: "cached", Gallery: []wire.MediaAsset{}, Contacts: []wire.PlaceContact{}, Hashtags: []wire.PlaceTag{}, Badges: []wire.Badge{}, InfoBlocks: []wire.PlaceInfoBlock{}}}
	repo := &repoStub{detailVersion: &now, detail: func() *PlaceRecord { value := validRecord(); return &value }()}
	got, err := testService(repo, &cacheStub{detail: &Cached[wire.PlaceDetailsResponse]{ETag: `"cached"`, Payload: cached}}).Detail(context.Background(), "valid-slug")
	if err != nil || got.Payload.Place.ID != "cached" || repo.detailCalls != 0 {
		t.Fatalf("cache result=%#v err=%v calls=%d", got, err, repo.detailCalls)
	}

	repo = &repoStub{detailVersion: &now, detail: func() *PlaceRecord { value := validRecord(); return &value }()}
	cache := &cacheStub{getErr: errors.New("redis down"), setErr: errors.New("redis down")}
	got, err = testService(repo, cache).Detail(context.Background(), "valid-slug")
	if err != nil || got.Payload.Place.Slug != "sulakskiy-kanon" || repo.detailCalls != 1 || cache.sets != 1 {
		t.Fatalf("fallback result=%#v err=%v calls=%d sets=%d", got, err, repo.detailCalls, cache.sets)
	}
}

func TestDetailBranchesUseStableErrors(t *testing.T) {
	now := time.Now().UTC()
	bad := validRecord()
	bad.NavigationPoint = []byte(`{"type":"Polygon","coordinates":[]}`)
	tests := []struct {
		name, want string
		repo       *repoStub
	}{
		{"version missing", "place_not_found", &repoStub{}},
		{"version failure", "internal_error", &repoStub{detailVersionErr: errors.New("version failed")}},
		{"lookup failure", "internal_error", &repoStub{detailVersion: &now, detailErr: errors.New("lookup failed")}},
		{"mapping failure", "internal_error", &repoStub{detailVersion: &now, detail: &bad}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := testService(test.repo, &cacheStub{}).Detail(context.Background(), "valid-slug")
			var appErr *AppError
			if !errors.As(err, &appErr) || appErr.Code != test.want {
				t.Fatalf("error=%#v want code=%s", err, test.want)
			}
		})
	}
}

func TestPublishAndArchiveReturnNotFoundForMissingRows(t *testing.T) {
	service := testService(&repoStub{}, &cacheStub{})
	for name, call := range map[string]func() error{
		"publish": func() error {
			_, err := service.Publish(context.Background(), "id", Actor{}, AuditEnvelope{})
			return err
		},
		"archive": func() error {
			_, err := service.Archive(context.Background(), "id", Actor{}, AuditEnvelope{})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			var appErr *AppError
			if err := call(); !errors.As(err, &appErr) || appErr.Code != "place_not_found" {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestPublishMapsFullAdminRecord(t *testing.T) {
	record := validRecord()
	record.Status = "published"
	repo := &repoStub{published: &record}
	got, err := testService(repo, &cacheStub{}).Publish(context.Background(), record.ID, Actor{}, AuditEnvelope{})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if got.ID != record.ID || got.Status != "published" || got.NavigationPoint == nil {
		t.Fatalf("unexpected admin view: %#v", got)
	}
}

func TestArchiveMapsFullAdminRecordAndLifecycleErrorsPassThrough(t *testing.T) {
	record := validRecord()
	record.Status = "archived"
	repo := &repoStub{archived: &record}
	got, err := testService(repo, &cacheStub{}).Archive(context.Background(), record.ID, Actor{}, AuditEnvelope{})
	if err != nil || got.ID != record.ID || got.Status != "archived" || repo.archiveID != record.ID {
		t.Fatalf("view=%#v err=%v archiveID=%q", got, err, repo.archiveID)
	}

	want := CannotPublishWithoutGallery()
	_, err = testService(&repoStub{publishErr: want}, &cacheStub{}).Publish(context.Background(), record.ID, Actor{}, AuditEnvelope{})
	if !errors.Is(err, want) {
		t.Fatalf("publish error=%#v, want original error", err)
	}

	bad := record
	bad.NavigationPoint = []byte(`{"type":"Polygon","coordinates":[]}`)
	for name, call := range map[string]func() error{
		"publish": func() error {
			_, err := testService(&repoStub{published: &bad}, &cacheStub{}).Publish(context.Background(), bad.ID, Actor{}, AuditEnvelope{})
			return err
		},
		"archive": func() error {
			_, err := testService(&repoStub{archived: &bad}, &cacheStub{}).Archive(context.Background(), bad.ID, Actor{}, AuditEnvelope{})
			return err
		},
	} {
		t.Run(name+" mapping failure", func(t *testing.T) {
			var appErr *AppError
			if err := call(); !errors.As(err, &appErr) || appErr.Code != "internal_error" {
				t.Fatalf("error=%#v", err)
			}
		})
	}
}
