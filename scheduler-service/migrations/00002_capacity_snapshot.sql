-- Scheduler's local, read-only copy of the dealership data booking needs:
-- hubs and their opening hours, service bays, and the service catalog
-- (architecture-design.md §4 option 2, §5b). dealership-service stays the
-- source of truth; once it exists, its events keep these tables current.
-- Until then they're filled by the dev seed (postgres/seed/init-scheduler.sql).
-- Ids are dealership-service's, so they're plain keys, not identities.

-- +goose Up
CREATE TABLE IF NOT EXISTS hub (
    id         BIGINT PRIMARY KEY,           -- dealership.id
    name       varchar(255) NOT NULL,
    address    varchar(255) NOT NULL,
    open_hour  time NOT NULL DEFAULT '08:00',
    close_hour time NOT NULL DEFAULT '18:00',
    -- IANA zone the opening hours are in; dealership-service doesn't model
    -- one yet.
    timezone   varchar(64) NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
    active     boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (close_hour > open_hour)
);

CREATE TABLE IF NOT EXISTS hub_bay (
    id         BIGINT PRIMARY KEY,           -- service_bay.id
    hub_id     BIGINT NOT NULL REFERENCES hub(id) ON DELETE CASCADE,
    code       varchar(50) NOT NULL,
    -- Only AVAILABLE bays take new bookings.
    status     varchar(20) NOT NULL DEFAULT 'AVAILABLE',
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hub_bay_hub ON hub_bay(hub_id);

CREATE TABLE IF NOT EXISTS catalog_service (
    id               BIGINT PRIMARY KEY,     -- service.id
    name             varchar(255) NOT NULL,
    service_type     varchar(255) NOT NULL,
    fee              numeric(12,2) NOT NULL CHECK (fee >= 0),
    duration_minutes int NOT NULL CHECK (duration_minutes > 0),
    active           boolean NOT NULL DEFAULT true,
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS catalog_service;
DROP TABLE IF EXISTS hub_bay;
DROP TABLE IF EXISTS hub;
