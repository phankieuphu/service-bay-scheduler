-- =============================================================
-- customer_db — customer-service
-- =============================================================
-- From ServiceHub.sql `customer`, with password_hash removed —
-- authentication is identity-service's job now (see
-- 01-identity-service.sql). This table is profile data only.
-- =============================================================

\connect customer_db

CREATE TABLE customer (
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
