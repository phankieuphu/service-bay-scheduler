-- =============================================================
-- scheduler_db — scheduler-service
-- =============================================================
-- From ServiceHub.sql §6 — the booking lifecycle. This is the
-- database architecture-design.md §4 is about: dealership_id,
-- customer_id, service_id, do_by (technician_id) and
-- customer_vehicle_id all point at other services' ids and lose
-- their FKs (Pattern a) — but bay_id/technician_id + the time
-- range are exactly the columns the no-double-booking exclusion
-- constraints need, and both stay right here on appointment_bay /
-- appointment_technician. Splitting the database does not weaken
-- that guarantee: the constraint never needed another service's
-- schema, only its own two columns.
--
-- appointment_bay_id on appointment_service keeps its real FK —
-- appointment_bay lives in this same database.
-- =============================================================

\connect scheduler_db

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE appointment (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dealership_id   BIGINT NOT NULL,    -- owned by dealership-service, no FK
    customer_id     BIGINT NOT NULL,    -- owned by customer-service, no FK
    scheduled_start timestamptz NOT NULL,
    scheduled_end   timestamptz NOT NULL,
    status          varchar(20) NOT NULL DEFAULT 'BOOKED'
                    CHECK (status IN ('BOOKED','CHECKED_IN','IN_PROGRESS',
                                      'COMPLETED','CANCELLED','NO_SHOW')),
    -- Scheduler's own running estimate, shown to the customer before
    -- the booking is billed. The authoritative bill — warranty fee +
    -- service fees + material cost — is billing-service's `bill` row,
    -- generated on ServiceCompleted (see 06-billing-service.sql).
    total_amount    numeric(12,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (scheduled_end > scheduled_start)
);

CREATE INDEX idx_appointment_dealership ON appointment(dealership_id, scheduled_start);
CREATE INDEX idx_appointment_customer   ON appointment(customer_id, scheduled_start);

-- -------------------------------------------------------------
-- ONE-TO-MANY: one appointment -> many service bays.
-- Each row is one bay booked for one time window inside one
-- appointment.
-- -------------------------------------------------------------
CREATE TABLE appointment_bay (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id BIGINT NOT NULL REFERENCES appointment(id) ON DELETE CASCADE,
    service_bay_id BIGINT NOT NULL,    -- owned by dealership-service, no FK
    start_time     timestamptz NOT NULL,
    end_time       timestamptz NOT NULL,
    status         varchar(20) NOT NULL DEFAULT 'RESERVED'
                   CHECK (status IN ('RESERVED','OCCUPIED','RELEASED','CANCELLED')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (end_time > start_time),

    -- generated range column, used by the overlap rule below
    slot tstzrange GENERATED ALWAYS AS (tstzrange(start_time, end_time, '[)')) STORED
);

CREATE INDEX idx_appt_bay_appointment ON appointment_bay(appointment_id);
CREATE INDEX idx_appt_bay_bay         ON appointment_bay(service_bay_id, start_time);

-- A bay cannot be booked by two appointments at the same time.
-- Cancelled rows are ignored. service_bay_id is a plain column with
-- no FK to dealership_db, but the constraint only needs equality on
-- that column plus this table's own slot range — no cross-database
-- lookup required (architecture-design.md §4).
ALTER TABLE appointment_bay
    ADD CONSTRAINT no_bay_double_booking
    EXCLUDE USING gist (service_bay_id WITH =, slot WITH &&)
    WHERE (status <> 'CANCELLED');

-- -------------------------------------------------------------
-- Vehicles brought in for this appointment (also one-to-many)
-- -------------------------------------------------------------
CREATE TABLE appointment_vehicle (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id      BIGINT NOT NULL REFERENCES appointment(id) ON DELETE CASCADE,
    customer_vehicle_id BIGINT NOT NULL,  -- owned by vehicle-service, no FK
    odometer_km         int CHECK (odometer_km >= 0),
    status              varchar(20) NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','RECEIVED','RETURNED')),
    UNIQUE (appointment_id, customer_vehicle_id)
);

CREATE INDEX idx_appt_vehicle_appointment ON appointment_vehicle(appointment_id);
CREATE INDEX idx_appt_vehicle_vehicle     ON appointment_vehicle(customer_vehicle_id);

-- -------------------------------------------------------------
-- Services performed. fee_charged is a snapshot of
-- dealership-service's service.fee at booking time (Pattern c) so
-- raising a price later does not change old invoices.
-- -------------------------------------------------------------
CREATE TABLE appointment_service (
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

CREATE INDEX idx_appt_service_appointment ON appointment_service(appointment_id);
CREATE INDEX idx_appt_service_service     ON appointment_service(service_id);
CREATE INDEX idx_appt_service_technician  ON appointment_service(do_by);
CREATE INDEX idx_appt_service_bay         ON appointment_service(appointment_bay_id);

-- -------------------------------------------------------------
-- Technician assignment with its own time window, so one
-- technician can't be booked twice at the same moment.
-- -------------------------------------------------------------
CREATE TABLE appointment_technician (
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

CREATE INDEX idx_appt_tech_appointment ON appointment_technician(appointment_id);
CREATE INDEX idx_appt_tech_technician  ON appointment_technician(technician_id, start_time);

ALTER TABLE appointment_technician
    ADD CONSTRAINT no_technician_double_booking
    EXCLUDE USING gist (technician_id WITH =, slot WITH &&)
    WHERE (status <> 'CANCELLED');

-- =============================================================
-- Helper view: which bays this service currently has reserved
-- =============================================================
-- ServiceHub.sql's original v_bay_current_usage joined service_bay
-- (dealership_db) straight to appointment_bay — a cross-database
-- join, which no longer works once they're separate databases.
--
-- The half of that view scheduler_db can still answer on its own —
-- which bay ids are busy, and for what window — stays here. The
-- human-readable half (the bay's `code`, its dealership) is
-- dealership-service's data; composing the two into one
-- "bay X at hub Y is busy until 3pm" response is a gateway/BFF job
-- (architecture-design.md §5, §7), not a database view.
-- =============================================================
CREATE VIEW v_appointment_bay_usage AS
SELECT ab.service_bay_id,
       ab.appointment_id,
       ab.start_time,
       ab.end_time
FROM appointment_bay ab
WHERE ab.status IN ('RESERVED','OCCUPIED')
  AND now() <@ ab.slot;
