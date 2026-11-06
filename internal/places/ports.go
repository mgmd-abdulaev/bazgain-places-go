package places

import (
	"context"
	"time"

	"github.com/mgmd-abdulaev/bazgain-places-go/internal/wire"
)

type Repository interface {
	CollectionsVersion(context.Context) (*time.Time, error)
	ListPublishedCollections(context.Context, int) ([]CollectionRecord, error)
	PlacesVersion(context.Context) (*time.Time, error)
	ListPublishedPlaces(context.Context, int) ([]PlaceRecord, error)
	DetailVersion(context.Context, string) (*time.Time, error)
	FindPublishedPlaceBySlug(context.Context, string) (*PlaceRecord, error)
	Publish(context.Context, string, Actor, AuditEnvelope) (*PlaceRecord, error)
	Archive(context.Context, string, Actor, AuditEnvelope) (*PlaceRecord, error)
}

type Cache interface {
	GetCollections(context.Context, *time.Time) (*Cached[wire.PlaceCollectionsResponse], error)
	SetCollections(context.Context, wire.PlaceCollectionsResponse, *time.Time) error
	GetPlaceList(context.Context, *time.Time) (*Cached[wire.PlaceListResponse], error)
	SetPlaceList(context.Context, wire.PlaceListResponse, *time.Time) error
	GetPlaceDetail(context.Context, string, *time.Time) (*Cached[wire.PlaceDetailsResponse], error)
	SetPlaceDetail(context.Context, string, wire.PlaceDetailsResponse, *time.Time) error
	BuildETag(*time.Time) string
}

type Cached[T any] struct {
	ETag    string
	Payload T
}

type Result[T any] struct {
	ETag    string
	Payload T
}
