-- =============================================================
-- dealership_db — dealership-service
-- =============================================================
-- From ServiceHub.sql §1, §2, §4, §5 — hubs, bays, storehouses,
-- materials/inventory, the service catalog, skills, and
-- technicians. Everything here is genuinely local to this service:
-- no table in this file references an id minted by another service,
-- so every FK below is a real FK.
--
-- inventory_transaction.appointment_id is the one exception — it
-- points at scheduler-service's `appointment`. The original
-- ServiceHub.sql added a real FK for it once `appointment` existed;
-- that FK is dropped here and kept as a plain column (Pattern a).
--
-- Fixed one bug from ServiceHub.sql while copying this table over:
-- `storehouse.id` was declared `GENERATED ALWAYS AS 3` (a typo for
-- `GENERATED ALWAYS AS IDENTITY`, which would have failed to run).
-- =============================================================

\connect dealership_db

-- -------------------------------------------------------------
-- 1. Dealership and physical resources
-- -------------------------------------------------------------

CREATE TABLE dealership (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          varchar(255) NOT NULL,
    address       varchar(255) NOT NULL,
    lat           numeric(9,6),
    lng           numeric(9,6),
    open_hour     time NOT NULL DEFAULT '08:00',
    close_hour    time NOT NULL DEFAULT '18:00',
    active_status varchar(20) NOT NULL DEFAULT 'ACTIVE'
                  CHECK (active_status IN ('ACTIVE','INACTIVE','CLOSED')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (close_hour > open_hour)
);

CREATE TABLE service_bay (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dealership_id BIGINT NOT NULL REFERENCES dealership(id) ON DELETE RESTRICT,
    code          varchar(50) NOT NULL,          -- e.g. "BAY-01"
    status        varchar(20) NOT NULL DEFAULT 'AVAILABLE'
                  CHECK (status IN ('AVAILABLE','MAINTENANCE','DISABLED')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (dealership_id, code)
);
-- Who is using a bay right now is derived from scheduler_db's
-- appointment_bay — see 05-scheduler-service.sql's note on
-- v_appointment_bay_usage for why that view can no longer live here.

CREATE INDEX idx_service_bay_dealership ON service_bay(dealership_id);

CREATE TABLE storehouse (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dealership_id BIGINT NOT NULL REFERENCES dealership(id) ON DELETE RESTRICT,
    name          varchar(255) NOT NULL,
    status        varchar(20) NOT NULL DEFAULT 'ACTIVE'
                  CHECK (status IN ('ACTIVE','INACTIVE')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (dealership_id, name)
);

CREATE INDEX idx_storehouse_dealership ON storehouse(dealership_id);

-- -------------------------------------------------------------
-- 2. Materials and inventory
-- -------------------------------------------------------------

CREATE TABLE material (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       varchar(255) NOT NULL,
    sku        varchar(100) NOT NULL UNIQUE,
    type       varchar(50)  NOT NULL,
    in_price   numeric(12,2) NOT NULL CHECK (in_price  >= 0),
    out_price  numeric(12,2) NOT NULL CHECK (out_price >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Current stock level: one row per (material, storehouse)
CREATE TABLE inventory_stock (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    material_id  BIGINT NOT NULL REFERENCES material(id)   ON DELETE RESTRICT,
    storage_id   BIGINT NOT NULL REFERENCES storehouse(id) ON DELETE RESTRICT,
    quantity     int NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (material_id, storage_id)
);

CREATE INDEX idx_inventory_stock_material ON inventory_stock(material_id);
CREATE INDEX idx_inventory_stock_storage  ON inventory_stock(storage_id);

-- Movement log: every IN / OUT / ADJUST is one row. Never updated.
-- This is the table technical-requirement.md §3 projects at
-- ~100-130M rows and range-partitions by month once volume demands it.
CREATE TABLE inventory_transaction (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    material_id    BIGINT NOT NULL REFERENCES material(id)   ON DELETE RESTRICT,
    storage_id     BIGINT NOT NULL REFERENCES storehouse(id) ON DELETE RESTRICT,
    direction      varchar(10) NOT NULL CHECK (direction IN ('IN','OUT','ADJUST')),
    quantity       int NOT NULL CHECK (quantity <> 0),
    unit_price     numeric(12,2) NOT NULL CHECK (unit_price >= 0),
    appointment_id BIGINT,          -- owned by scheduler-service, no FK (Pattern a)
    note           varchar(255),
    occurred_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_inv_txn_material     ON inventory_transaction(material_id, occurred_at);
CREATE INDEX idx_inv_txn_storage      ON inventory_transaction(storage_id, occurred_at);
CREATE INDEX idx_inv_txn_appointment  ON inventory_transaction(appointment_id);

-- -------------------------------------------------------------
-- 3. Services and skills
-- -------------------------------------------------------------

CREATE TABLE service_type (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name varchar(255) NOT NULL UNIQUE
);

CREATE TABLE service (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_type_id  BIGINT NOT NULL REFERENCES service_type(id) ON DELETE RESTRICT,
    name             varchar(255) NOT NULL,
    fee              numeric(12,2) NOT NULL CHECK (fee >= 0),
    duration_minutes int NOT NULL DEFAULT 60 CHECK (duration_minutes > 0),
    active           boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_service_type ON service(service_type_id);

CREATE TABLE skill (
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name      varchar(255) NOT NULL UNIQUE,
    max_level smallint NOT NULL DEFAULT 5 CHECK (max_level BETWEEN 1 AND 10)
);

-- Which skill (and what level) a service requires
CREATE TABLE service_skill (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_id     BIGINT NOT NULL REFERENCES service(id) ON DELETE CASCADE,
    skill_id       BIGINT NOT NULL REFERENCES skill(id)   ON DELETE RESTRICT,
    required_level smallint NOT NULL CHECK (required_level >= 1),
    UNIQUE (service_id, skill_id)
);

CREATE INDEX idx_service_skill_service ON service_skill(service_id);
CREATE INDEX idx_service_skill_skill   ON service_skill(skill_id);

-- Materials a service normally consumes (the "recipe")
CREATE TABLE service_material (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_id  BIGINT NOT NULL REFERENCES service(id)  ON DELETE CASCADE,
    material_id BIGINT NOT NULL REFERENCES material(id) ON DELETE RESTRICT,
    count       int NOT NULL CHECK (count > 0),
    UNIQUE (service_id, material_id)
);

CREATE INDEX idx_service_material_service  ON service_material(service_id);
CREATE INDEX idx_service_material_material ON service_material(material_id);

-- -------------------------------------------------------------
-- 4. Technicians
-- -------------------------------------------------------------

CREATE TABLE technician (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dealership_id BIGINT NOT NULL REFERENCES dealership(id) ON DELETE RESTRICT,
    name          varchar(255) NOT NULL,
    level         varchar(20) NOT NULL DEFAULT 'JUNIOR'
                  CHECK (level IN ('APPRENTICE','JUNIOR','SENIOR','MASTER')),
    active_status varchar(20) NOT NULL DEFAULT 'ACTIVE'
                  CHECK (active_status IN ('ACTIVE','ON_LEAVE','RESIGNED')),
    join_date     date NOT NULL,
    birthday      date,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_technician_dealership ON technician(dealership_id);

CREATE TABLE technician_skill (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    technician_id BIGINT NOT NULL REFERENCES technician(id) ON DELETE CASCADE,
    skill_id      BIGINT NOT NULL REFERENCES skill(id)      ON DELETE RESTRICT,
    skill_level   smallint NOT NULL CHECK (skill_level >= 0),
    UNIQUE (technician_id, skill_id)
);

CREATE INDEX idx_tech_skill_technician ON technician_skill(technician_id);
CREATE INDEX idx_tech_skill_skill      ON technician_skill(skill_id);
