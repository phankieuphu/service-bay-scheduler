-- =============================================================
-- billing_db — billing-service
-- =============================================================
-- Not present in ServiceHub.sql — that schema only had
-- appointment.total_amount and appointment_service.fee_charged,
-- both living in scheduler_db. architecture-design.md §1a flagged
-- billing as "implied by §6 but had no owner"; this is that owner.
--
-- warranty_fee / service_fee / material_cost are Pattern c copies:
-- resolved once, from vehicle-service (warranty status) and
-- dealership-service (fee_charged sum, material cost), at the
-- moment `ServiceCompleted` fires — not re-derived from a live
-- vehicle_id lookup on every read. That's what keeps a bill from
-- one month silently changing because a warranty expired the next.
-- The three-part total matches business-requirement.md §6 exactly.
-- =============================================================

\connect billing_db

CREATE TABLE bill (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id BIGINT NOT NULL UNIQUE,  -- owned by scheduler-service, no FK
    customer_id    BIGINT NOT NULL,         -- owned by customer-service, no FK
    vehicle_id     BIGINT NOT NULL,         -- owned by vehicle-service, no FK
    warranty_fee   numeric(12,2) NOT NULL DEFAULT 0 CHECK (warranty_fee  >= 0),
    service_fee    numeric(12,2) NOT NULL DEFAULT 0 CHECK (service_fee   >= 0),
    material_cost  numeric(12,2) NOT NULL DEFAULT 0 CHECK (material_cost >= 0),
    total_amount   numeric(12,2) GENERATED ALWAYS AS
                       (warranty_fee + service_fee + material_cost) STORED,
    status         varchar(20) NOT NULL DEFAULT 'PENDING'
                   CHECK (status IN ('PENDING','PARTIALLY_PAID','PAID','VOID')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_bill_customer ON bill(customer_id);

-- A deposit at booking time and the final payment are both rows
-- here; `bill.status` reflects whatever the sum of SUCCEEDED
-- payments covers.
CREATE TABLE payment (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bill_id    BIGINT NOT NULL REFERENCES bill(id) ON DELETE RESTRICT,
    type       varchar(20) NOT NULL CHECK (type IN ('DEPOSIT','FINAL')),
    method     varchar(30) NOT NULL,
    amount     numeric(12,2) NOT NULL CHECK (amount > 0),
    status     varchar(20) NOT NULL DEFAULT 'PENDING'
               CHECK (status IN ('PENDING','SUCCEEDED','FAILED','REFUNDED')),
    paid_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_bill ON payment(bill_id);
