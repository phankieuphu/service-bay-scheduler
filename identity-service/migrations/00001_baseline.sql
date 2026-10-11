-- Baseline: the schema postgres/init/01-identity-service.sql used to create.
-- Every statement is IF NOT EXISTS, so a database that init script already
-- built (an existing volume) adopts this migration instead of failing on
-- it, and picks up anything it was missing. Later changes go in new,
-- numbered files: never edit a migration that has shipped.

-- +goose Up
-- =============================================================
-- identity_db — identity-service
-- =============================================================
-- Not present in ServiceHub.sql — that schema stored password_hash
-- directly on `customer`, with no login path for technicians or
-- dealership managers at all. architecture-design.md §1a already
-- flagged identity as "implied by §2 but had no owner"; this is
-- that owner. Every other service trusts this service's tokens
-- and stops storing credentials itself.
--
-- The login identity comes first: registering here writes a
-- UserCreated event to identity.user-events, and customer-service /
-- vehicle-service create their local profile from it, storing
-- app_user.id as a plain reference (Pattern a, no FK). So this
-- table holds no pointer to those profiles — they don't exist yet
-- when the row is inserted.
--
-- id is a UUID minted by identity-service (architecture-design.md
-- §5a), so it's unambiguous in every service that stores it.
-- =============================================================

CREATE TABLE IF NOT EXISTS app_user (
    id            uuid PRIMARY KEY,
    email         varchar(255) NOT NULL,     -- stored lowercased
    password_hash varchar(255) NOT NULL,     -- bcrypt, never plain text
    role          varchar(20) NOT NULL
                  CHECK (role IN ('CUSTOMER','TECHNICIAN','MANAGER','ADMIN')),
    status        varchar(20) NOT NULL DEFAULT 'ACTIVE'
                  CHECK (status IN ('ACTIVE','INACTIVE','LOCKED')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness, and the index the login lookup
-- (WHERE lower(email) = lower($1)) uses.
CREATE UNIQUE INDEX IF NOT EXISTS uq_app_user_email ON app_user (lower(email));

-- Transactional outbox: a row is inserted here in the same transaction as
-- the app_user write it describes, then relayed to Kafka by a poller
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

-- +goose Down
DROP TABLE IF EXISTS outbox_message;
DROP TABLE IF EXISTS app_user;
