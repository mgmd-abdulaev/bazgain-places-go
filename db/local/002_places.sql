-- IMPORTANT: This migration creates raw SQL constraints that Prisma cannot
-- represent. Do NOT run `prisma db pull` after this migration — it will lose
-- the raw constraints. Forward-fix only via additional migrations.
-- See plan §1.4a, §1.6, §1.7.
--
-- Notes:
--   - PostGIS GIST indexes on `places.navigation_point` / `places.geometry`
--     are intentionally NOT created in Phase 1 (no spatial filters yet).
--     Add CONCURRENTLY in a follow-up migration when a spatial query lands.
--   - The cross-collection invariant on `place_collection_filter_memberships`
--     (a filter from collection A cannot tag a membership in collection B)
--     is enforced by app-level guards in `place-collections.admin.service.ts`,
--     NOT by a composite FK — composite FKs survive the schema but Prisma's
--     `db pull` drops them, which would be a silent regression.

-- CreateEnum
CREATE TYPE "place_status" AS ENUM ('draft', 'published', 'archived');

-- CreateEnum
CREATE TYPE "place_media_kind" AS ENUM ('image', 'video');

-- CreateEnum
CREATE TYPE "place_contact_kind" AS ENUM ('phone', 'telegram', 'whatsapp', 'website');

-- CreateEnum
CREATE TYPE "place_badge_tone" AS ENUM ('neutral', 'accent', 'success', 'warning');

-- CreateEnum
CREATE TYPE "place_info_block_type" AS ENUM ('single_line', 'large');

-- CreateEnum
CREATE TYPE "place_day_of_week" AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');

-- CreateEnum
CREATE TYPE "place_collection_status" AS ENUM ('draft', 'published', 'archived');

-- CreateTable
CREATE TABLE "place_kinds" (
    "id" UUID NOT NULL,
    "key" VARCHAR(80) NOT NULL,
    "label" VARCHAR(120) NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_kinds_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_tags" (
    "id" UUID NOT NULL,
    "slug" VARCHAR(80) NOT NULL,
    "title" VARCHAR(40) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "info_blocks" (
    "id" UUID NOT NULL,
    "type" "place_info_block_type" NOT NULL,
    "icon_key" VARCHAR(80) NOT NULL,
    "title" VARCHAR(60) NOT NULL,
    "value_markdown" TEXT NOT NULL,
    "background_color_light" VARCHAR(9),
    "background_color_dark" VARCHAR(9),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "info_blocks_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "places" (
    "id" UUID NOT NULL,
    "slug" VARCHAR(120) NOT NULL,
    "title" VARCHAR(120) NOT NULL,
    "kind_id" UUID NOT NULL,
    "summary" VARCHAR(280),
    "description_markdown" TEXT,
    "status" "place_status" NOT NULL DEFAULT 'draft',
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "location_label" VARCHAR(120),
    "address_display_text" VARCHAR(240),
    "address_region" VARCHAR(120),
    "address_district" VARCHAR(120),
    "address_locality" VARCHAR(120),
    "address_street" VARCHAR(160),
    "address_house" VARCHAR(40),
    "opening_hours_timezone" VARCHAR(60),
    "opening_hours_note_markdown" TEXT,
    "cover_media_id" UUID,
    "navigation_point" geometry(Point, 4326),
    "geometry" geometry(Geometry, 4326),
    "published_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "places_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_media" (
    "id" UUID NOT NULL,
    "place_id" UUID NOT NULL,
    "kind" "place_media_kind" NOT NULL,
    "url" TEXT NOT NULL,
    "thumbnail_url" TEXT,
    "width" INTEGER,
    "height" INTEGER,
    "alt_text" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_media_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_tag_assignments" (
    "place_id" UUID NOT NULL,
    "tag_id" UUID NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "place_tag_assignments_pkey" PRIMARY KEY ("place_id","tag_id")
);

-- CreateTable
CREATE TABLE "place_badges" (
    "id" UUID NOT NULL,
    "place_id" UUID NOT NULL,
    "title" VARCHAR(40) NOT NULL,
    "tone" "place_badge_tone" NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_badges_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_contacts" (
    "id" UUID NOT NULL,
    "place_id" UUID NOT NULL,
    "type" "place_contact_kind" NOT NULL,
    "label" VARCHAR(40),
    "display_value" TEXT NOT NULL,
    "action_url" TEXT NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_contacts_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_opening_periods" (
    "id" UUID NOT NULL,
    "place_id" UUID NOT NULL,
    "open_day" "place_day_of_week" NOT NULL,
    "open_minutes" SMALLINT NOT NULL,
    "close_day" "place_day_of_week" NOT NULL,
    "close_minutes" SMALLINT NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_opening_periods_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_special_opening_periods" (
    "id" UUID NOT NULL,
    "place_id" UUID NOT NULL,
    "start_date" DATE NOT NULL,
    "end_date" DATE,
    "closed" BOOLEAN NOT NULL,
    "open_minutes" SMALLINT,
    "close_minutes" SMALLINT,
    "note_markdown" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_special_opening_periods_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_info_block_assignments" (
    "place_id" UUID NOT NULL,
    "info_block_id" UUID NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "place_info_block_assignments_pkey" PRIMARY KEY ("place_id","info_block_id")
);

-- CreateTable
CREATE TABLE "place_collections" (
    "id" UUID NOT NULL,
    "slug" VARCHAR(120) NOT NULL,
    "title" VARCHAR(120) NOT NULL,
    "subtitle" VARCHAR(200),
    "description" TEXT,
    "preview_image_url" TEXT,
    "icon_image_url" TEXT,
    "background_color" VARCHAR(9),
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "status" "place_collection_status" NOT NULL DEFAULT 'draft',
    "published_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_collections_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_collection_filters" (
    "id" UUID NOT NULL,
    "collection_id" UUID NOT NULL,
    "slug" VARCHAR(120) NOT NULL,
    "title" VARCHAR(120) NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "place_collection_filters_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_collection_memberships" (
    "id" UUID NOT NULL,
    "place_id" UUID NOT NULL,
    "collection_id" UUID NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "place_collection_memberships_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "place_collection_filter_memberships" (
    "id" UUID NOT NULL,
    "membership_id" UUID NOT NULL,
    "filter_id" UUID NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "place_collection_filter_memberships_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "place_kinds_key_key" ON "place_kinds"("key");

-- CreateIndex
CREATE UNIQUE INDEX "place_tags_slug_key" ON "place_tags"("slug");

-- CreateIndex
CREATE INDEX "places_status_sort_order_updated_at_idx" ON "places"("status", "sort_order", "updated_at" DESC);

-- CreateIndex
CREATE INDEX "places_status_updated_at_id_idx" ON "places"("status", "updated_at" DESC, "id" DESC);

-- CreateIndex
CREATE INDEX "places_kind_id_idx" ON "places"("kind_id");

-- CreateIndex
CREATE UNIQUE INDEX "places_slug_key" ON "places"("slug");

-- CreateIndex
CREATE INDEX "place_media_place_id_sort_order_idx" ON "place_media"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_tag_assignments_place_id_sort_order_idx" ON "place_tag_assignments"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_badges_place_id_sort_order_idx" ON "place_badges"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_contacts_place_id_sort_order_idx" ON "place_contacts"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_opening_periods_place_id_sort_order_idx" ON "place_opening_periods"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_special_opening_periods_place_id_sort_order_idx" ON "place_special_opening_periods"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_info_block_assignments_place_id_sort_order_idx" ON "place_info_block_assignments"("place_id", "sort_order");

-- CreateIndex
CREATE INDEX "place_collections_status_sort_order_updated_at_idx" ON "place_collections"("status", "sort_order", "updated_at" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "place_collections_slug_key" ON "place_collections"("slug");

-- CreateIndex
CREATE INDEX "place_collection_filters_collection_id_sort_order_idx" ON "place_collection_filters"("collection_id", "sort_order");

-- CreateIndex
CREATE UNIQUE INDEX "place_collection_filters_collection_id_slug_key" ON "place_collection_filters"("collection_id", "slug");

-- CreateIndex
CREATE INDEX "place_collection_memberships_collection_id_sort_order_idx" ON "place_collection_memberships"("collection_id", "sort_order");

-- CreateIndex
CREATE UNIQUE INDEX "place_collection_memberships_place_id_collection_id_key" ON "place_collection_memberships"("place_id", "collection_id");

-- CreateIndex
CREATE INDEX "place_collection_filter_memberships_filter_id_sort_order_idx" ON "place_collection_filter_memberships"("filter_id", "sort_order");

-- CreateIndex
CREATE UNIQUE INDEX "place_collection_filter_memberships_membership_id_filter_id_key" ON "place_collection_filter_memberships"("membership_id", "filter_id");

-- AddForeignKey
ALTER TABLE "places" ADD CONSTRAINT "places_kind_id_fkey" FOREIGN KEY ("kind_id") REFERENCES "place_kinds"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "places" ADD CONSTRAINT "places_cover_media_id_fkey" FOREIGN KEY ("cover_media_id") REFERENCES "place_media"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_media" ADD CONSTRAINT "place_media_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_tag_assignments" ADD CONSTRAINT "place_tag_assignments_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_tag_assignments" ADD CONSTRAINT "place_tag_assignments_tag_id_fkey" FOREIGN KEY ("tag_id") REFERENCES "place_tags"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_badges" ADD CONSTRAINT "place_badges_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_contacts" ADD CONSTRAINT "place_contacts_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_opening_periods" ADD CONSTRAINT "place_opening_periods_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_special_opening_periods" ADD CONSTRAINT "place_special_opening_periods_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_info_block_assignments" ADD CONSTRAINT "place_info_block_assignments_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_info_block_assignments" ADD CONSTRAINT "place_info_block_assignments_info_block_id_fkey" FOREIGN KEY ("info_block_id") REFERENCES "info_blocks"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_collection_filters" ADD CONSTRAINT "place_collection_filters_collection_id_fkey" FOREIGN KEY ("collection_id") REFERENCES "place_collections"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_collection_memberships" ADD CONSTRAINT "place_collection_memberships_place_id_fkey" FOREIGN KEY ("place_id") REFERENCES "places"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_collection_memberships" ADD CONSTRAINT "place_collection_memberships_collection_id_fkey" FOREIGN KEY ("collection_id") REFERENCES "place_collections"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_collection_filter_memberships" ADD CONSTRAINT "place_collection_filter_memberships_membership_id_fkey" FOREIGN KEY ("membership_id") REFERENCES "place_collection_memberships"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "place_collection_filter_memberships" ADD CONSTRAINT "place_collection_filter_memberships_filter_id_fkey" FOREIGN KEY ("filter_id") REFERENCES "place_collection_filters"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- ─── Slug / key / icon-key shape guards (mirrors src/common/validation/patterns.ts) ───

ALTER TABLE "places"
  ADD CONSTRAINT "places_slug_shape"
    CHECK ("slug" ~ '^[a-z0-9]([a-z0-9-]{0,118}[a-z0-9])?$');

ALTER TABLE "place_tags"
  ADD CONSTRAINT "place_tags_slug_shape"
    CHECK ("slug" ~ '^[a-z0-9]([a-z0-9-]{0,118}[a-z0-9])?$');

ALTER TABLE "place_collections"
  ADD CONSTRAINT "place_collections_slug_shape"
    CHECK ("slug" ~ '^[a-z0-9]([a-z0-9-]{0,118}[a-z0-9])?$');

ALTER TABLE "place_collection_filters"
  ADD CONSTRAINT "place_collection_filters_slug_shape"
    CHECK ("slug" ~ '^[a-z0-9]([a-z0-9-]{0,118}[a-z0-9])?$');

ALTER TABLE "place_kinds"
  ADD CONSTRAINT "place_kinds_key_shape"
    CHECK ("key" ~ '^[a-z0-9]([a-z0-9_-]{0,78}[a-z0-9])?$');

ALTER TABLE "info_blocks"
  ADD CONSTRAINT "info_blocks_icon_key_shape"
    CHECK ("icon_key" ~ '^[a-z0-9_-]{1,80}$');

-- ─── HTTPS-only URLs (mobile WebView / iOS ATS won't load http://) ───

ALTER TABLE "place_media"
  ADD CONSTRAINT "place_media_url_https"
    CHECK ("url" LIKE 'https://%'),
  ADD CONSTRAINT "place_media_thumbnail_url_https"
    CHECK ("thumbnail_url" IS NULL OR "thumbnail_url" LIKE 'https://%');

ALTER TABLE "place_collections"
  ADD CONSTRAINT "place_collections_preview_image_url_https"
    CHECK ("preview_image_url" IS NULL OR "preview_image_url" LIKE 'https://%'),
  ADD CONSTRAINT "place_collections_icon_image_url_https"
    CHECK ("icon_image_url" IS NULL OR "icon_image_url" LIKE 'https://%');

-- ─── Hex color shape + light/dark pairing for info_blocks ───

ALTER TABLE "place_collections"
  ADD CONSTRAINT "place_collections_background_color_hex"
    CHECK ("background_color" IS NULL OR "background_color" ~ '^#[0-9A-Fa-f]{6}([0-9A-Fa-f]{2})?$');

ALTER TABLE "info_blocks"
  ADD CONSTRAINT "info_blocks_background_color_light_hex"
    CHECK ("background_color_light" IS NULL OR "background_color_light" ~ '^#[0-9A-Fa-f]{6}([0-9A-Fa-f]{2})?$'),
  ADD CONSTRAINT "info_blocks_background_color_dark_hex"
    CHECK ("background_color_dark" IS NULL OR "background_color_dark" ~ '^#[0-9A-Fa-f]{6}([0-9A-Fa-f]{2})?$'),
  ADD CONSTRAINT "info_blocks_color_pair"
    CHECK (("background_color_light" IS NULL) = ("background_color_dark" IS NULL));

-- ─── Contacts: action_url must be a known scheme prefix ───

ALTER TABLE "place_contacts"
  ADD CONSTRAINT "place_contacts_action_url_scheme"
    CHECK ("action_url" ~ '^(https://|tel:|mailto:)');

-- ─── Address consistency: display_text required when any structured part is set ───

ALTER TABLE "places"
  ADD CONSTRAINT "places_address_consistency"
    CHECK (
      (
        "address_region" IS NULL
        AND "address_district" IS NULL
        AND "address_locality" IS NULL
        AND "address_street" IS NULL
        AND "address_house" IS NULL
      )
      OR "address_display_text" IS NOT NULL
    );

-- ─── Publish gate: published place must have a navigation point ───

ALTER TABLE "places"
  ADD CONSTRAINT "places_navigation_point_required_when_published"
    CHECK ("status" <> 'published' OR "navigation_point" IS NOT NULL);

-- ─── PostGIS geometry validity (last-resort guard; DTO validators are the
--      primary defence — see places module GeoJSON validators in Phase 4). ───

ALTER TABLE "places"
  ADD CONSTRAINT "places_geometry_valid"
    CHECK ("geometry" IS NULL OR (ST_IsValid("geometry") AND ST_SRID("geometry") = 4326)),
  ADD CONSTRAINT "places_navigation_point_valid"
    CHECK ("navigation_point" IS NULL OR (ST_IsValid("navigation_point") AND ST_SRID("navigation_point") = 4326));

-- ─── description_markdown length cap (mirrors MAX_PLACE_DESCRIPTION_LENGTH=16000) ───

ALTER TABLE "places"
  ADD CONSTRAINT "places_description_length"
    CHECK ("description_markdown" IS NULL OR LENGTH("description_markdown") <= 16000);

-- ─── Opening hours minute ranges. SMALLINT can hold negative numbers and
--      values up to 32k — narrow it down to the [0, 1440] day window. 1440
--      means "24:00" (end-of-day boundary, used by intervals that close at
--      midnight; mapper translates to "24:00" on the wire). ───

ALTER TABLE "place_opening_periods"
  ADD CONSTRAINT "place_opening_periods_minutes_range"
    CHECK (
      "open_minutes" BETWEEN 0 AND 1440
      AND "close_minutes" BETWEEN 0 AND 1440
    );

-- ─── Special periods: closed=true requires both minute fields NULL,
--      closed=false requires both NOT NULL. Range-check the optional
--      minutes too. Date order: end_date >= start_date when set. ───

ALTER TABLE "place_special_opening_periods"
  ADD CONSTRAINT "place_special_opening_periods_closed_pairing"
    CHECK (
      ("closed" = true AND "open_minutes" IS NULL AND "close_minutes" IS NULL)
      OR ("closed" = false AND "open_minutes" IS NOT NULL AND "close_minutes" IS NOT NULL)
    ),
  ADD CONSTRAINT "place_special_opening_periods_minutes_range"
    CHECK (
      ("open_minutes" IS NULL OR "open_minutes" BETWEEN 0 AND 1440)
      AND ("close_minutes" IS NULL OR "close_minutes" BETWEEN 0 AND 1440)
    ),
  ADD CONSTRAINT "place_special_opening_periods_date_order"
    CHECK ("end_date" IS NULL OR "end_date" >= "start_date");
