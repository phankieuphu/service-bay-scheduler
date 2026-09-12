-- =============================================================
-- identity_db — identity-service
-- =============================================================
-- Not present in ServiceHub.sql — that schema stored password_hash
-- directly on `customer`, with no login path for technicians or
-- dealership managers at all. architecture-design.md §1a already
-- flagged identity as "implied by §2 but had no owner"; this is
-- that owner. Every other service trusts this service's tokens
-- and stops storing credentials itself.
-- =============================================================

\connect identity_db

CREATE TABLE app_user (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_type    varchar(20) NOT NULL
                  CHECK (actor_type IN ('CUSTOMER','TECHNICIAN','MANAGER','ADMIN')),
    actor_ref_id  BIGINT NOT NULL,     -- customer.id / technician.id in the owning
                                        -- service's own database — Pattern a, no FK
    email         varchar(255) NOT NULL UNIQUE,
    password_hash varchar(255) NOT NULL,   -- bcrypt/argon2, never plain text
    role          varchar(30) NOT NULL DEFAULT 'ACTOR',
    status        varchar(20) NOT NULL DEFAULT 'ACTIVE'
                  CHECK (status IN ('ACTIVE','INACTIVE','LOCKED')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    -- one login identity per (actor_type, actor_ref_id): a technician
    -- and a customer could otherwise collide on the same numeric id
    -- since those ids are minted independently by different services
    UNIQUE (actor_type, actor_ref_id)
);

CREATE INDEX idx_app_user_actor ON app_user(actor_type, actor_ref_id);
