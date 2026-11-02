package places

import (
	"context"
	"log/slog"
	"regexp"

	"github.com/tx-mi/bazgain-places-go/internal/wire"
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,118}[a-z0-9])?$`)
	uuidShape   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

type Service struct {
	repo          Repository
	cache         Cache
	logger        *slog.Logger
	placeCap      int
	collectionCap int
}

func NewService(repo Repository, cache Cache, logger *slog.Logger, placeCap, collectionCap int) *Service {
	return &Service{repo: repo, cache: cache, logger: logger, placeCap: placeCap, collectionCap: collectionCap}
}

func (s *Service) Collections(ctx context.Context) (Result[wire.PlaceCollectionsResponse], error) {
	version, err := s.repo.CollectionsVersion(ctx)
	if err != nil {
		return Result[wire.PlaceCollectionsResponse]{}, Internal(err)
	}
	etag := s.cache.BuildETag(version)
	if cached, cacheErr := s.cache.GetCollections(ctx, version); cacheErr == nil && cached != nil {
		return Result[wire.PlaceCollectionsResponse]{ETag: cached.ETag, Payload: cached.Payload}, nil
	}
	if version == nil {
		payload := wire.PlaceCollectionsResponse{Collections: []wire.PlaceCollectionSummary{}}
		_ = s.cache.SetCollections(ctx, payload, version)
		return Result[wire.PlaceCollectionsResponse]{ETag: etag, Payload: payload}, nil
	}
	rows, err := s.repo.ListPublishedCollections(ctx, s.collectionCap)
	if err != nil {
		return Result[wire.PlaceCollectionsResponse]{}, Internal(err)
	}
	if len(rows) >= s.collectionCap*8/10 {
		s.logger.WarnContext(ctx, "public collections list approaching hard cap", "count", len(rows), "hard_cap", s.collectionCap)
	}
	items := make([]wire.PlaceCollectionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, MapCollection(row))
	}
	payload := wire.PlaceCollectionsResponse{Collections: items}
	_ = s.cache.SetCollections(ctx, payload, version)
	return Result[wire.PlaceCollectionsResponse]{ETag: etag, Payload: payload}, nil
}

func (s *Service) List(ctx context.Context) (Result[wire.PlaceListResponse], error) {
	version, err := s.repo.PlacesVersion(ctx)
	if err != nil {
		return Result[wire.PlaceListResponse]{}, Internal(err)
	}
	etag := s.cache.BuildETag(version)
	if cached, cacheErr := s.cache.GetPlaceList(ctx, version); cacheErr == nil && cached != nil {
		return Result[wire.PlaceListResponse]{ETag: cached.ETag, Payload: cached.Payload}, nil
	}
	if version == nil {
		payload := wire.PlaceListResponse{Items: []wire.PlaceSummary{}}
		_ = s.cache.SetPlaceList(ctx, payload, version)
		return Result[wire.PlaceListResponse]{ETag: etag, Payload: payload}, nil
	}
	rows, err := s.repo.ListPublishedPlaces(ctx, s.placeCap)
	if err != nil {
		return Result[wire.PlaceListResponse]{}, Internal(err)
	}
	if len(rows) >= s.placeCap*8/10 {
		s.logger.WarnContext(ctx, "public places list approaching hard cap", "count", len(rows), "hard_cap", s.placeCap)
	}
	items := make([]wire.PlaceSummary, 0, len(rows))
	for _, row := range rows {
		mapped, mapErr := MapSummary(row)
		if mapErr != nil {
			return Result[wire.PlaceListResponse]{}, Internal(mapErr)
		}
		items = append(items, mapped)
	}
	payload := wire.PlaceListResponse{Items: items}
	_ = s.cache.SetPlaceList(ctx, payload, version)
	return Result[wire.PlaceListResponse]{ETag: etag, Payload: payload}, nil
}

func (s *Service) Detail(ctx context.Context, slug string) (Result[wire.PlaceDetailsResponse], error) {
	if uuidShape.MatchString(slug) || !slugPattern.MatchString(slug) {
		return Result[wire.PlaceDetailsResponse]{}, NotFound()
	}
	version, err := s.repo.DetailVersion(ctx, slug)
	if err != nil {
		return Result[wire.PlaceDetailsResponse]{}, Internal(err)
	}
	if version == nil {
		return Result[wire.PlaceDetailsResponse]{}, NotFound()
	}
	etag := s.cache.BuildETag(version)
	if cached, cacheErr := s.cache.GetPlaceDetail(ctx, slug, version); cacheErr == nil && cached != nil {
		return Result[wire.PlaceDetailsResponse]{ETag: cached.ETag, Payload: cached.Payload}, nil
	}
	row, err := s.repo.FindPublishedPlaceBySlug(ctx, slug)
	if err != nil {
		return Result[wire.PlaceDetailsResponse]{}, Internal(err)
	}
	if row == nil {
		return Result[wire.PlaceDetailsResponse]{}, NotFound()
	}
	mapped, mapErr := MapDetails(*row)
	if mapErr != nil {
		return Result[wire.PlaceDetailsResponse]{}, Internal(mapErr)
	}
	payload := wire.PlaceDetailsResponse{Place: mapped}
	_ = s.cache.SetPlaceDetail(ctx, slug, payload, version)
	return Result[wire.PlaceDetailsResponse]{ETag: etag, Payload: payload}, nil
}

func (s *Service) Publish(ctx context.Context, id string, actor Actor, env AuditEnvelope) (wire.PlaceAdminView, error) {
	row, err := s.repo.Publish(ctx, id, actor, env)
	if err != nil {
		return wire.PlaceAdminView{}, err
	}
	if row == nil {
		return wire.PlaceAdminView{}, NotFound()
	}
	view, mapErr := MapAdmin(*row)
	if mapErr != nil {
		return wire.PlaceAdminView{}, Internal(mapErr)
	}
	return view, nil
}
func (s *Service) Archive(ctx context.Context, id string, actor Actor, env AuditEnvelope) (wire.PlaceAdminView, error) {
	row, err := s.repo.Archive(ctx, id, actor, env)
	if err != nil {
		return wire.PlaceAdminView{}, err
	}
	if row == nil {
		return wire.PlaceAdminView{}, NotFound()
	}
	view, mapErr := MapAdmin(*row)
	if mapErr != nil {
		return wire.PlaceAdminView{}, Internal(mapErr)
	}
	return view, nil
}
