-- =============================================================
-- vehicle_db — vehicle-service
-- =============================================================
-- From ServiceHub.sql §3. `customer_vehicle.customer_id` and
-- `vehicle_material.material_id` cross into customer-service and
-- dealership-service respectively — those FKs are dropped and kept
-- as plain columns (Pattern a, architecture-design.md §5): validated
-- once at write time by the caller, never re-checked after.
-- Everything that stays inside this database (vehicle_model,
-- vehicle, the current-owner uniqueness rule) keeps its real FK,
-- because a real FK is still correct — and still enforced — for
-- relationships that don't cross a service boundary.
-- =============================================================

\connect vehicle_db

CREATE TABLE vehicle_model (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    make       varchar(100) NOT NULL,
    model      varchar(100) NOT NULL,
    year       smallint CHECK (year BETWEEN 1900 AND 2100),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (make, model, year)
);

CREATE TABLE vehicle (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vin               varchar(17) NOT NULL UNIQUE,
    license_plate     varchar(20),
    vehicle_model_id  BIGINT NOT NULL REFERENCES vehicle_model(id) ON DELETE RESTRICT,
    warranty_end_date date,
    status            varchar(20) NOT NULL DEFAULT 'ACTIVE'
                      CHECK (status IN ('ACTIVE', 'SOLD','SCRAPPED')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

Create Index Idx_Vehicle_Model On Vehicle(Vehicle_Model_Id);

-- Ownership can change over time
CREATE TABLE customer_vehicle (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id BIGINT NOT NULL,     -- owned by customer-service, no FK
    vehicle_id  BIGINT NOT NULL REFERENCES vehicle(id) ON DELETE RESTRICT,
    owned_from  date NOT NULL DEFAULT CURRENT_DATE,
    owned_to    date,
    status      varchar(20) NOT NULL DEFAULT 'CURRENT'
                CHECK (status IN ('CURRENT','TRANSFERRED')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (owned_to IS NULL OR owned_to >= owned_from)
);

CREATE INDEX idx_cust_vehicle_customer ON customer_vehicle(customer_id);
CREATE INDEX idx_cust_vehicle_vehicle  ON customer_vehicle(vehicle_id);

-- Only one CURRENT owner per vehicle
CREATE UNIQUE INDEX uq_vehicle_current_owner
    ON customer_vehicle(vehicle_id) WHERE status = 'CURRENT';

-- Parts installed on a specific car
CREATE TABLE vehicle_material (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vehicle_id   BIGINT NOT NULL REFERENCES vehicle(id) ON DELETE CASCADE,
    material_id  BIGINT NOT NULL,    -- owned by dealership-service, no FK
    count        int NOT NULL CHECK (count > 0),
    description  varchar(255),
    installed_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (vehicle_id, material_id)
);

CREATE INDEX idx_veh_material_vehicle  ON vehicle_material(vehicle_id);
CREATE INDEX idx_veh_material_material ON vehicle_material(material_id);


CREATE TABLE outbox_message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        varchar(255) NOT NULL,
    message_key  varchar(255) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

-- Partial index so the relay's poll (WHERE published_at IS NULL) stays
-- cheap regardless of how many published rows have piled up.
CREATE INDEX idx_outbox_message_unpublished ON outbox_message (id) WHERE published_at IS NULL;
