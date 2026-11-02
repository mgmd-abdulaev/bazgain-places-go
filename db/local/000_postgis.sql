-- Enable PostGIS. Must run before any geometry() column is declared.
CREATE EXTENSION IF NOT EXISTS postgis;

-- Infra-only probe for exercising Prisma Unsupported("geometry(Point, 4326)")
-- round-trips. Intentionally without a GIST index (Phase 0 decision; probe
-- is not queried by anything).
CREATE TABLE "_postgis_probe" (
    "id"       UUID NOT NULL,
    "location" geometry(Point, 4326) NOT NULL,

    CONSTRAINT "_postgis_probe_pkey" PRIMARY KEY ("id")
);
