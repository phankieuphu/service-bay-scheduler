-- Baseline: the schema postgres/init/02-customer-service.sql used to create.
-- Every statement is IF NOT EXISTS, so a database that init script already
-- built (an existing volume) adopts this migration instead of failing on
-- it, and picks up anything it was missing. Later changes go in new,
-- numbered files: never edit a migration that has shipped.

-- +goose Up
-- =============================================================
-- customer_db — customer-service
-- =============================================================
-- From ServiceHub.sql `customer`, with password_hash removed —
-- authentication is identity-service's job now (see
-- 01-identity-service.sql). This table is profile data only.
-- =============================================================

CREATE TABLE IF NOT EXISTS customer (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       varchar(255) NOT NULL,
    email      varchar(255) NOT NULL UNIQUE,
    phone      varchar(30),
    birthday   date,
    status     varchar(20) NOT NULL DEFAULT 'ACTIVE'
               CHECK (status IN ('ACTIVE','INACTIVE','BANNED')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Transactional outbox: a row is inserted here in the same transaction as
-- the customer write it describes, then relayed to Kafka by a poller
-- (see internal/adapters/kafka.OutboxRelay) — see architecture-design.md
-- §"Every service also writes an outbox row...".
CREATE TABLE IF NOT EXISTS outbox_message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        varchar(255) NOT NULL,
    message_key  varchar(255) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

-- Partial index so the relay's poll (WHERE published_at IS NULL) stays
-- cheap regardless of how many published rows have piled up.
CREATE INDEX IF NOT EXISTS idx_outbox_message_unpublished ON outbox_message (id) WHERE published_at IS NULL;

ALTER TABLE customer ADD COLUMN IF NOT EXISTS delete_at timestamptz DEFAULT NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_message;
DROP TABLE IF EXISTS customer;
