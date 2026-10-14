-- Baseline: the schema postgres/init/05-scheduler-service.sql used to
-- create. Every statement is IF NOT EXISTS (or guarded), so a database that
-- init script already built (an existing volume) adopts this migration
-- instead of failing on it. Later changes go in new, numbered files: never
-- edit a migration that has shipped.
--
-- dealership_id, customer_id, service_id, do_by (technician_id) and
-- customer_vehicle_id all point at other services' ids and have no FKs
-- (architecture-design.md §5a). bay/technician id + time range are the
-- columns the no-double-booking exclusion constraints need, and both stay
-- here on appointment_bay / appointment_technician (§4).

-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS appointment (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dealership_id   BIGINT NOT NULL,    -- owned by dealership-service, no FK
    customer_id     BIGINT NOT NULL,    -- owned by customer-service, no FK
    scheduled_start timestamptz NOT NULL,
    scheduled_end   timestamptz NOT NULL,
    status          varchar(20) NOT NULL DEFAULT 'BOOKED'
                    CHECK (status IN ('BOOKED','CHECKED_IN','IN_PROGRESS',
                                      'COMPLETED','CANCELLED','NO_SHOW')),
    -- Scheduler's own running estimate, shown to the customer before the
    -- booking is billed. The authoritative bill is billing-service's.
    total_amount    numeric(12,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (scheduled_end > scheduled_start)
);

CREATE INDEX IF NOT EXISTS idx_appointment_dealership ON appointment(dealership_id, scheduled_start);
CREATE INDEX IF NOT EXISTS idx_appointment_customer   ON appointment(customer_id, scheduled_start);

-- One appointment -> many service bays, each for one time window.
CREATE TABLE IF NOT EXISTS appointment_bay (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id BIGINT NOT NULL REFERENCES appointment(id) ON DELETE CASCADE,
    service_bay_id BIGINT NOT NULL,    -- owned by dealership-service, no FK
    start_time     timestamptz NOT NULL,
    end_time       timestamptz NOT NULL,
    status         varchar(20) NOT NULL DEFAULT 'RESERVED'
                   CHECK (status IN ('RESERVED','OCCUPIED','RELEASED','CANCELLED')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (end_time > start_time),
    slot tstzrange GENERATED ALWAYS AS (tstzrange(start_time, end_time, '[)')) STORED
);

CREATE INDEX IF NOT EXISTS idx_appt_bay_appointment ON appointment_bay(appointment_id);
CREATE INDEX IF NOT EXISTS idx_appt_bay_bay         ON appointment_bay(service_bay_id, start_time);

-- A bay cannot be booked by two appointments at the same time. Cancelled
-- rows are ignored.
-- +goose StatementBegin
DO $$
BEGIN
    PERFORM 1 FROM pg_constraint WHERE conname = 'no_bay_double_booking';
    IF NOT FOUND THEN
        ALTER TABLE appointment_bay
            ADD CONSTRAINT no_bay_double_booking
            EXCLUDE USING gist (service_bay_id WITH =, slot WITH &&)
            WHERE (status <> 'CANCELLED');
    END IF;
END $$;
-- +goose StatementEnd

-- Vehicles brought in for this appointment.
CREATE TABLE IF NOT EXISTS appointment_vehicle (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id      BIGINT NOT NULL REFERENCES appointment(id) ON DELETE CASCADE,
    customer_vehicle_id BIGINT NOT NULL,  -- owned by vehicle-service, no FK
    odometer_km         int CHECK (odometer_km >= 0),
    status              varchar(20) NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','RECEIVED','RETURNED')),
    UNIQUE (appointment_id, customer_vehicle_id)
);

CREATE INDEX IF NOT EXISTS idx_appt_vehicle_appointment ON appointment_vehicle(appointment_id);
CREATE INDEX IF NOT EXISTS idx_appt_vehicle_vehicle     ON appointment_vehicle(customer_vehicle_id);

-- Services performed. fee_charged is a snapshot of the catalog fee at
-- booking time (§5c), so a later price change doesn't rewrite old bookings.
CREATE TABLE IF NOT EXISTS appointment_service (
    id                     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id         BIGINT NOT NULL REFERENCES appointment(id) ON DELETE CASCADE,
    service_id             BIGINT NOT NULL,   -- owned by dealership-service, no FK
    appointment_vehicle_id BIGINT REFERENCES appointment_vehicle(id) ON DELETE SET NULL,
    appointment_bay_id     BIGINT REFERENCES appointment_bay(id)     ON DELETE SET NULL,
    do_by                  BIGINT,            -- technician.id, owned by dealership-service, no FK
    fee_charged            numeric(12,2) NOT NULL CHECK (fee_charged >= 0),
    started_at             timestamptz,
    finished_at            timestamptz,
    status                 varchar(20) NOT NULL DEFAULT 'PENDING'
                           CHECK (status IN ('PENDING','IN_PROGRESS','DONE','CANCELLED')),
    CHECK (finished_at IS NULL OR started_at IS NULL OR finished_at >= started_at)
);

CREATE INDEX IF NOT EXISTS idx_appt_service_appointment ON appointment_service(appointment_id);
CREATE INDEX IF NOT EXISTS idx_appt_service_service     ON appointment_service(service_id);
CREATE INDEX IF NOT EXISTS idx_appt_service_technician  ON appointment_service(do_by);
CREATE INDEX IF NOT EXISTS idx_appt_service_bay         ON appointment_service(appointment_bay_id);

-- Technician assignment with its own time window, so one technician can't
-- be booked twice at the same moment.
CREATE TABLE IF NOT EXISTS appointment_technician (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id BIGINT NOT NULL REFERENCES appointment(id) ON DELETE CASCADE,
    technician_id  BIGINT NOT NULL,   -- owned by dealership-service, no FK
    start_time     timestamptz NOT NULL,
    end_time       timestamptz NOT NULL,
    status         varchar(20) NOT NULL DEFAULT 'ASSIGNED'
                   CHECK (status IN ('ASSIGNED','WORKING','DONE','CANCELLED')),
    CHECK (end_time > start_time),
    slot tstzrange GENERATED ALWAYS AS (tstzrange(start_time, end_time, '[)')) STORED
);

CREATE INDEX IF NOT EXISTS idx_appt_tech_appointment ON appointment_technician(appointment_id);
CREATE INDEX IF NOT EXISTS idx_appt_tech_technician  ON appointment_technician(technician_id, start_time);

-- +goose StatementBegin
DO $$
BEGIN
    PERFORM 1 FROM pg_constraint WHERE conname = 'no_technician_double_booking';
    IF NOT FOUND THEN
        ALTER TABLE appointment_technician
            ADD CONSTRAINT no_technician_double_booking
            EXCLUDE USING gist (technician_id WITH =, slot WITH &&)
            WHERE (status <> 'CANCELLED');
    END IF;
END $$;
-- +goose StatementEnd

-- Which bays are reserved right now. The human-readable half (bay code,
-- hub) is dealership-service's data, so this view stays id-only.
CREATE OR REPLACE VIEW v_appointment_bay_usage AS
SELECT ab.service_bay_id,
       ab.appointment_id,
       ab.start_time,
       ab.end_time
FROM appointment_bay ab
WHERE ab.status IN ('RESERVED','OCCUPIED')
  AND now() <@ ab.slot;

-- Transactional outbox: a row is inserted here in the same transaction as
-- the appointment write it describes, then relayed to Kafka by a poller
-- (internal/adapters/kafka.OutboxRelay).
CREATE TABLE IF NOT EXISTS outbox_message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        varchar(255) NOT NULL,
    message_key  varchar(255) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_outbox_message_unpublished ON outbox_message (id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_message;
DROP VIEW IF EXISTS v_appointment_bay_usage;
DROP TABLE IF EXISTS appointment_technician;
DROP TABLE IF EXISTS appointment_service;
DROP TABLE IF EXISTS appointment_vehicle;
DROP TABLE IF EXISTS appointment_bay;
DROP TABLE IF EXISTS appointment;
