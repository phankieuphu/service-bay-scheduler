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

-- Itemized lines behind bill's three totals, written by POST
-- /api/v1/bills/checkout in the same transaction as the bill. Copied
-- at checkout time (Pattern c, same reason as the totals above), so a
-- later catalog or price change never rewrites an issued bill.
-- Bills created with plain POST /api/v1/bills have no lines.
CREATE TABLE bill_line (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bill_id     BIGINT NOT NULL REFERENCES bill(id) ON DELETE CASCADE,
    kind        varchar(20) NOT NULL CHECK (kind IN ('WARRANTY_FEE','SERVICE','MATERIAL')),
    description varchar(255) NOT NULL,
    quantity    integer NOT NULL CHECK (quantity > 0),
    unit_price  numeric(12,2) NOT NULL CHECK (unit_price >= 0),
    amount      numeric(12,2) NOT NULL CHECK (amount >= 0),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_bill_line_bill ON bill_line(bill_id);

-- A deposit at booking time and the final payment are both rows
-- here; `bill.status` reflects whatever the sum of SUCCEEDED
-- payments covers.
--
-- CASH payments are SUCCEEDED as soon as they are recorded. CARD
-- payments start PENDING and are settled by payment-service (Stripe):
-- external_payment_id is payment-service's payment id (no FK, other
-- database), and the row moves to SUCCEEDED/FAILED when the matching
-- payment.payment.* event arrives. A PENDING row counts against the
-- outstanding balance so two concurrent card checkouts can't overpay.
CREATE TABLE payment (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bill_id             BIGINT NOT NULL REFERENCES bill(id) ON DELETE RESTRICT,
    type                varchar(20) NOT NULL CHECK (type IN ('DEPOSIT','FINAL')),
    method              varchar(30) NOT NULL CHECK (method IN ('CASH','CARD')),
    amount              numeric(12,2) NOT NULL CHECK (amount > 0),
    status              varchar(20) NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','SUCCEEDED','FAILED','REFUNDED')),
    external_payment_id BIGINT UNIQUE,      -- owned by payment-service, no FK
    failure_reason      varchar(500),
    paid_at             timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_bill ON payment(bill_id);

-- Transactional outbox, same shape and relay contract as the Go
-- services (see 01-identity-service.sql). Carries PaymentReceived.
CREATE TABLE outbox_message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        varchar(255) NOT NULL,
    message_key  varchar(255) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX idx_outbox_message_unpublished ON outbox_message (id) WHERE published_at IS NULL;
