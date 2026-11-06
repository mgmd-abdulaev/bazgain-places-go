package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/places"
)

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	txTimeout    time.Duration
}

func New(pool *pgxpool.Pool, queryTimeout, txTimeout time.Duration) *Repository {
	return &Repository{pool: pool, queryTimeout: queryTimeout, txTimeout: txTimeout}
}

func (r *Repository) withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.queryTimeout)
}

func (r *Repository) CollectionsVersion(ctx context.Context) (*time.Time, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var value *time.Time
	err := r.pool.QueryRow(ctx, `SELECT MAX(updated_at) FROM place_collections WHERE status='published'::place_collection_status`).Scan(&value)
	return value, err
}

func (r *Repository) ListPublishedCollections(ctx context.Context, limit int) ([]places.CollectionRecord, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT c.id::text,c.slug,c.title,c.subtitle,c.description,c.preview_image_url,c.icon_image_url,c.background_color,c.sort_order,COUNT(m.id)::int FROM place_collections c LEFT JOIN place_collection_memberships m ON m.collection_id=c.id WHERE c.status='published'::place_collection_status GROUP BY c.id ORDER BY c.sort_order ASC,c.updated_at DESC,c.id ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]places.CollectionRecord, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var item places.CollectionRecord
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &item.Subtitle, &item.Description, &item.PreviewImageURL, &item.IconImageURL, &item.BackgroundColor, &item.SortOrder, &item.PlacesCount); err != nil {
			return nil, err
		}
		result = append(result, item)
		ids = append(ids, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return result, nil
	}
	filters, err := r.pool.Query(ctx, `SELECT f.id::text,f.collection_id::text,f.slug,f.title,f.sort_order,COUNT(fm.id)::int FROM place_collection_filters f LEFT JOIN place_collection_filter_memberships fm ON fm.filter_id=f.id WHERE f.collection_id=ANY($1::uuid[]) GROUP BY f.id ORDER BY f.collection_id,f.sort_order,f.id`, ids)
	if err != nil {
		return nil, err
	}
	defer filters.Close()
	byID := make(map[string]*places.CollectionRecord, len(result))
	for i := range result {
		byID[result[i].ID] = &result[i]
		result[i].Filters = []places.CollectionFilterRecord{}
	}
	for filters.Next() {
		var item places.CollectionFilterRecord
		var collectionID string
		if err := filters.Scan(&item.ID, &collectionID, &item.Slug, &item.Title, &item.SortOrder, &item.PlacesCount); err != nil {
			return nil, err
		}
		if collection := byID[collectionID]; collection != nil {
			collection.Filters = append(collection.Filters, item)
		}
	}
	return result, filters.Err()
}

func (r *Repository) PlacesVersion(ctx context.Context) (*time.Time, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var value *time.Time
	err := r.pool.QueryRow(ctx, `SELECT MAX(updated_at) FROM places WHERE status='published'::place_status`).Scan(&value)
	return value, err
}
func (r *Repository) DetailVersion(ctx context.Context, slug string) (*time.Time, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var value *time.Time
	err := r.pool.QueryRow(ctx, `SELECT MAX(updated_at) FROM places WHERE slug=$1 AND status='published'::place_status`, slug).Scan(&value)
	return value, err
}

const placeBaseSelectPrefix = `SELECT p.id::text,p.slug,p.title,p.summary,p.description_markdown,p.status::text,p.sort_order,p.location_label,p.address_display_text,p.address_region,p.address_district,p.address_locality,p.address_street,p.address_house,p.opening_hours_timezone,p.opening_hours_note_markdown,p.cover_media_id::text,`
const placeBaseSelectSuffix = `,ST_AsGeoJSON(p.navigation_point)::jsonb,ST_AsGeoJSON(p.geometry)::jsonb,p.published_at,p.created_at,p.updated_at,k.id::text,k.key,k.label FROM places p JOIN place_kinds k ON k.id=p.kind_id LEFT JOIN place_media cover ON cover.id=p.cover_media_id LEFT JOIN LATERAL (SELECT pm.url FROM place_media pm WHERE pm.place_id=p.id AND pm.kind='image'::place_media_kind ORDER BY pm.sort_order,pm.id LIMIT 1) fallback ON TRUE `

const publicBaseSelect = placeBaseSelectPrefix + `COALESCE(cover.url,fallback.url)` + placeBaseSelectSuffix
const adminBaseSelect = placeBaseSelectPrefix + `cover.url` + placeBaseSelectSuffix

func scanPlace(row pgx.Row) (places.PlaceRecord, error) {
	var item places.PlaceRecord
	var nav, geom []byte
	err := row.Scan(&item.ID, &item.Slug, &item.Title, &item.Summary, &item.DescriptionMarkdown, &item.Status, &item.SortOrder, &item.LocationLabel, &item.AddressDisplayText, &item.AddressRegion, &item.AddressDistrict, &item.AddressLocality, &item.AddressStreet, &item.AddressHouse, &item.OpeningHoursTimezone, &item.OpeningHoursNoteMarkdown, &item.CoverMediaID, &item.CoverImageURL, &nav, &geom, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt, &item.Kind.ID, &item.Kind.Key, &item.Kind.Label)
	item.NavigationPoint = json.RawMessage(nav)
	item.Geometry = json.RawMessage(geom)
	return item, err
}

func (r *Repository) ListPublishedPlaces(ctx context.Context, limit int) ([]places.PlaceRecord, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, publicBaseSelect+` WHERE p.status='published'::place_status AND p.navigation_point IS NOT NULL ORDER BY p.sort_order,p.updated_at DESC,p.id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]places.PlaceRecord, 0)
	for rows.Next() {
		item, scanErr := scanPlace(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadListRelations(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) FindPublishedPlaceBySlug(ctx context.Context, slug string) (*places.PlaceRecord, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	item, err := scanPlace(r.pool.QueryRow(ctx, publicBaseSelect+` WHERE p.slug=$1 AND p.status='published'::place_status AND p.navigation_point IS NOT NULL`, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items := []places.PlaceRecord{item}
	if err := r.loadAllRelations(ctx, items, true); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *Repository) Publish(ctx context.Context, id string, actor places.Actor, env places.AuditEnvelope) (*places.PlaceRecord, error) {
	return r.mutateLifecycle(ctx, id, "published", actor, env)
}
func (r *Repository) Archive(ctx context.Context, id string, actor places.Actor, env places.AuditEnvelope) (*places.PlaceRecord, error) {
	return r.mutateLifecycle(ctx, id, "archived", actor, env)
}

func (r *Repository) mutateLifecycle(parent context.Context, id, status string, actor places.Actor, env places.AuditEnvelope) (*places.PlaceRecord, error) {
	ctx, cancel := context.WithTimeout(parent, r.txTimeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		rollbackTimeout := 2 * time.Second
		if r.txTimeout > 0 && r.txTimeout < rollbackTimeout {
			rollbackTimeout = r.txTimeout
		}
		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), rollbackTimeout)
		defer rollbackCancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var slug string
	var publishedAt *time.Time
	var hasNavigation, hasGallery bool
	err = tx.QueryRow(ctx, `SELECT p.slug,p.published_at,p.navigation_point IS NOT NULL,EXISTS(SELECT 1 FROM place_media pm WHERE pm.place_id=p.id) FROM places p WHERE p.id=$1::uuid FOR UPDATE`, id).Scan(&slug, &publishedAt, &hasNavigation, &hasGallery)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, places.NotFound()
	}
	if err != nil {
		return nil, err
	}
	if status == "published" && !hasGallery {
		return nil, places.CannotPublishWithoutGallery()
	}
	if status == "published" && !hasNavigation {
		return nil, places.CannotPublishWithoutNavigationPoint()
	}
	if status == "published" {
		_, err = tx.Exec(ctx, `UPDATE places SET status='published'::place_status,published_at=COALESCE(published_at,statement_timestamp()),updated_at=statement_timestamp() WHERE id=$1::uuid`, id)
	} else {
		_, err = tx.Exec(ctx, `UPDATE places SET status='archived'::place_status,updated_at=statement_timestamp() WHERE id=$1::uuid`, id)
	}
	if err != nil {
		return nil, err
	}
	auditID, auditErr := uuid.NewV7()
	if auditErr != nil {
		return nil, auditErr
	}
	event := "place_archived"
	meta := map[string]any{"place_id": id, "slug": slug}
	if status == "published" {
		event = "place_published"
		meta["first_publish"] = publishedAt == nil
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(id,event_type,user_id,session_id,ip,user_agent,meta,created_at) VALUES($1,$2,$3::uuid,$4::uuid,$5,$6,$7::jsonb,statement_timestamp())`, auditID.String(), event, actor.UserID, actor.SessionID, env.IP, env.UserAgent, string(metaJSON))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.findAdminByID(parent, id)
}

func (r *Repository) findAdminByID(ctx context.Context, id string) (*places.PlaceRecord, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	item, err := scanPlace(r.pool.QueryRow(ctx, adminBaseSelect+` WHERE p.id=$1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items := []places.PlaceRecord{item}
	if err := r.loadAllRelations(ctx, items, false); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func idsOf(items []places.PlaceRecord) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
func indexOf(items []places.PlaceRecord) map[string]*places.PlaceRecord {
	result := make(map[string]*places.PlaceRecord, len(items))
	for i := range items {
		result[items[i].ID] = &items[i]
	}
	return result
}
func wrapLoad(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("load %s: %w", name, err)
}
